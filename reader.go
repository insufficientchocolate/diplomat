package diplomat

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	yaml "gopkg.in/yaml.v2"
)

type PreprocessorConfig struct {
	Type    string
	Options YAMLOption
}

type OutputConfig struct {
	Selectors []string
	Templates []MessengerConfig
}

// Outline is the struct of translation file.
type Outline struct {
	Version       string
	Preprocessors []PreprocessorConfig
	Output        []OutputConfig
}

type PartialTranslation struct {
	path string
	data YAMLMap
}

func NewReader(dir string) *Reader {
	return &Reader{
		dir:                    dir,
		outlineChan:            make(chan *Outline),
		partialTranslationChan: make(chan *PartialTranslation),
		errChan:                make(chan error, 10),
	}
}

type Reader struct {
	dir                    string
	outlineChan            chan *Outline
	partialTranslationChan chan *PartialTranslation
	errChan                chan error
}

func (r Reader) GetOutlineSource() <-chan *Outline {
	return r.outlineChan
}

func (r Reader) GetPartialTranslationSource() <-chan *PartialTranslation {
	return r.partialTranslationChan
}

func (r Reader) GetErrorOut() <-chan error {
	return r.errChan
}

func (r Reader) pushError(e error) {
	go func() {
		select {
		case r.errChan <- e:
			return
		default:
			log.Println("an error drop by reader", e)
		}
	}()
}

func (r Reader) Read() (*Outline, []*PartialTranslation, error) {
	outlineChan, translationChan, errorChan := r.doRead(true)
	var wg sync.WaitGroup
	wg.Add(2)
	var outline *Outline
	go func() {
		outline = <-outlineChan
		wg.Done()
	}()
	translations := make([]*PartialTranslation, 0, 1)
	go func() {
		for t := range translationChan {
			translations = append(translations, t)
		}
		wg.Done()
	}()
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()
	for {
		select {
		case <-done:
			return outline, translations, nil
		case err := <-errorChan:
			return nil, nil, err
		}
	}
}

type asyncErrorSink struct {
	errorChan chan error
}

func (a asyncErrorSink) push(err error) {
	go func() {
		select {
		case a.errorChan <- err:
			return
		default:
			log.Println("[error-sink]an error dropped ", err)
		}
	}()
}

func newAsyncErrorSink() asyncErrorSink {
	return asyncErrorSink{
		errorChan: make(chan error),
	}
}

func (r Reader) doRead(closeChannels bool) (<-chan *Outline, <-chan *PartialTranslation, <-chan error) {
	outlineChan := make(chan *Outline)
	translationChan := make(chan *PartialTranslation)
	errorSink := newAsyncErrorSink()
	go func() {
		o, err := parseOutline(filepath.Join(r.dir, "diplomat.yaml"))
		if err != nil {
			errorSink.push(err)
			return
		}
		outlineChan <- o
		if closeChannels {
			close(outlineChan)
		}
	}()
	go func() {
		var wg sync.WaitGroup
		paths, err := filepath.Glob(filepath.Join(r.dir, "*.yaml"))
		if err != nil {
			errorSink.push(err)
			return
		}
		for _, p := range paths {
			if isOutlineFile(p) {
				continue
			}
			wg.Add(1)
			go func(path string) {
				t, err := parsePartialTranslation(path)
				if err != nil {
					r.pushError(err)
					return
				}
				translationChan <- t
				wg.Done()
			}(p)
		}
		wg.Wait()
		if closeChannels {
			close(translationChan)
		}
	}()
	return outlineChan, translationChan, errorSink.errorChan
}

func (r Reader) Watch() {
	r.Read()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		r.pushError(err)
	}
	watcher.Add(r.dir)
	for e := range nameBaseThrottler(watcher.Events) {
		if isOutlineFile(e.Name) {
			go func(path string) {
				o, err := parseOutline(path)
				if err != nil {
					r.pushError(err)
					return
				}
				r.outlineChan <- o
			}(e.Name)
		} else {
			go func(path string) {
				t, err := parsePartialTranslation(path)
				if err != nil {
					r.pushError(err)
					return
				}
				r.partialTranslationChan <- t
			}(e.Name)
		}
	}
}

func isOutlineFile(name string) bool {
	return strings.TrimRight(filepath.Base(name), filepath.Ext(name)) == "diplomat"
}

func parseOutline(name string) (*Outline, error) {
	data, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}
	var outline Outline
	err = yaml.Unmarshal(data, &outline)
	if err != nil {
		return nil, err
	}
	return &outline, nil
}

func parsePartialTranslation(path string) (*PartialTranslation, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t YAMLMap = make(YAMLMap)
	err = yaml.Unmarshal(data, &t)
	if err != nil {
		return nil, err
	}
	return &PartialTranslation{
		path: path,
		data: t,
	}, nil
}

type watchThrottler struct {
	source     <-chan fsnotify.Event
	out        chan<- fsnotify.Event
	throttlers map[string]chan<- fsnotify.Event
}

func (wt watchThrottler) loop() {
	for e := range wt.source {
		c, exist := wt.throttlers[e.Name]
		if !exist {
			nc := make(chan fsnotify.Event, 1)
			go func() {
				for e := range throttle(time.Second, nc) {
					wt.out <- e
				}
			}()
			wt.throttlers[e.Name] = nc
			c = nc
		}
		c <- e
	}
}

func (wt watchThrottler) close() {
	for _, c := range wt.throttlers {
		close(c)
	}
	close(wt.out)
}

func nameBaseThrottler(source <-chan fsnotify.Event) <-chan fsnotify.Event {
	c := make(chan fsnotify.Event, 1)
	w := watchThrottler{
		source,
		c,
		make(map[string]chan<- fsnotify.Event),
	}
	go w.loop()
	return c
}

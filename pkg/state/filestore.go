package state

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// listen for signals, and send stops to the generator
var (
	chQuit = make(chan os.Signal)
)

type filestore struct {
	key      string
	filename string
	states   map[string]*MsgState
}

func NewFilestore(key, filename string) SessionStore {
	filestore := &filestore{
		key:      key,
		filename: filename,
		states:   make(map[string]*MsgState),
	}
	signal.Notify(chQuit, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-chQuit:
			fmt.Println("Got signal, wrapping up")
			filestore.flushToDisk()
		}
	}()
	gob.Register(map[string]interface{}{})
	// gob.Register(MsgState{})
	return filestore
}

func (f *filestore) flushToDisk() error {
	b := new(bytes.Buffer)
	enc := gob.NewEncoder(b)
	err := enc.Encode(f.states)
	if err != nil {
		return err
	}

	fh, eopen := os.OpenFile(f.filename, os.O_CREATE|os.O_WRONLY, 0666)
	defer fh.Close()
	if eopen != nil {
		return eopen
	}
	n, e := fh.Write(b.Bytes())
	if e != nil {
		return e
	}
	fmt.Fprintf(os.Stderr, "%d bytes successfully written to file\n", n)
	return nil
}

func (f *filestore) Set(path string, state *MsgState) error {
	f.states[f.key+"-"+path] = state
	return f.flushToDisk()
}

func (f *filestore) Get(path string) (*MsgState, error) {
	currentState := f.states[f.key+"-"+path]

	if currentState == nil {
		fh, err := os.Open(f.filename)
		if err != nil {
			return currentState, err
		}
		states := make(map[string]*MsgState)
		dec := gob.NewDecoder(fh)
		err = dec.Decode(&states)
		if err != nil {
			return currentState, err
		}
		currentState = states[f.key+"-"+path]
	}
	return currentState, nil
}
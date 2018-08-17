package main

import (
	"io"
	"time"
	"sync"
	"sync/atomic"
)

type task struct {
	done chan struct{}
	src io.ReadCloser
	dst io.WriteCloser
	bytePerSecond float64
	err error
	startTime time.Time
	endTime time.Time
	mutex sync.Mutex
	readNum int64
	fileSize int64
	filename string
	buffer []byte
}

func (t *task) getReadNum() int64{
	if t == nil {
		return 0
	}
	return atomic.LoadInt64(&t.readNum)
}

func (t *task) start(){

	var rn,wn int

	go t.bps()

	t.startTime=time.Now()

loop:
	rn,err:=t.src.Read(t.buffer)

	if err!=nil||rn==0{
		goto done
	}

	wn,err=t.dst.Write(t.buffer[:rn])

	if err!=nil{
		goto done
	} else if rn!=wn {
		err = io.ErrShortWrite
		goto done
	}else{
		atomic.AddInt64(&t.readNum,int64(rn))
		goto loop
	}

done:
		t.err=err
		close(t.done)
		t.endTime=time.Now()
	return
}

func (t *task) bps(){
	var prev int64
	then := t.startTime

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-t.done:
			return

		case now := <-ticker.C:
			d := now.Sub(then)
			then = now

			cur := t.getReadNum()
			bs := cur - prev
			prev = cur

			t.mutex.Lock()
			t.bytePerSecond = float64(bs) / d.Seconds()
			t.mutex.Unlock()
		}
	}
}

func (t *task) getBps() float64{
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.bytePerSecond
}
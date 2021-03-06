package main

// The purpose of the work package is to show how you can use an unbuffered channel
// to create a pool of goroutines that will perform and control the amount of work that
// gets done concurrently. This is a better approach than using a buffered channel of
// some arbitrary static size that acts as a queue of work and throwing a bunch of goroutines at it.
// Unbuffered channels provide a guarantee that data has been exchanged
// between two goroutines. This approach of using an unbuffered channel allows the
// user to know when the pool is performing the work, and the channel pushes back
// when it can’t accept any more work because it’s busy. No work is ever lost or stuck in a
// queue that has no guarantee it will ever be worked on.

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
)

// Worker must be implemented by types that want to use
// the work pool.
// The Worker interface declares a single method called Task
type Worker interface {
	Task() error
}

// Pool provides a pool of goroutines that can execute any Worker
// tasks that are submitted.
// a struct named Pool is declared, which is the type that implements the
// pool of goroutines and will have methods that process the work. The type declares two
// fields, one named work, which is a channel of the Worker interface type, and a sync.WaitGroup named wg.
type Pool struct {
	work    chan Worker
	wg      sync.WaitGroup
	errChan chan error
}

// New creates a new work pool.
func New(maxGoroutines int) *Pool {
	p := Pool{
		work:    make(chan Worker),
		errChan: make(chan error),
	}
	p.wg.Add(maxGoroutines)
	// The for range loop blocks until there’s a Worker interface value to receive on the
	// work channel. When a value is received, the Task method is called. Once the work
	// channel is closed, the for range loop ends and the call to Done on the WaitGroup is
	// called. Then the goroutine terminates.
	for i := 0; i < maxGoroutines; i++ {
		go func() {
			for w := range p.work {
				p.errChan <- w.Task()
			}
			p.wg.Done()
		}()
	}
	return &p
}

// Run submits work to the pool.
// This method is used to submit work into the
// pool. It accepts an interface value of type Worker and sends that value through the
// work channel. Since the work channel is an unbuffered channel, the caller must wait
// for a goroutine from the pool to receive it. This is what we want, because the caller
// needs the guarantee that the work being submitted is being worked on once the call to Run returns.
func (p *Pool) Run(w Worker) (err error) {
	p.work <- w
	select {
	case err = <-p.errChan:
	}
	return
}

// Shutdown waits for all the goroutines to shutdown.
// The Shutdown method in listing 7.33 does two things. First, it closes the work channel, which causes all of the goroutines in the pool to shut down and call the Done
// method on the WaitGroup. Then the Shutdown method calls the Wait method on the
// WaitGroup, which causes the Shutdown method to wait for all the goroutines to report
// they have terminated.
func (p *Pool) Shutdown() {
	close(p.work)
	close(p.errChan)
	p.wg.Wait()
}

var names = []string{
	"steve",
	"bob",
	"mary",
	"jason",
	"Bob",
	"Lee",
	"Jane",
}

// namePrinter provides special support for printing names.
type namePrinter struct {
	name string
}

// Task implements the Worker interface.
func (m *namePrinter) Task() error {
	time.Sleep(time.Second * 1)
	if m.name == "jason" {
		return errors.New("Invalid name")
	}
	log.Println(m.name)
	return nil
}

// main is the entry point for all Go programs.
func main() {
	// Create a work pool with 2 goroutines.
	p := New(2)
	var wg sync.WaitGroup
	wg.Add(len(names))
	// Iterate over the slice of names.
	for _, name := range names {
		// Create a namePrinter and provide the
		// specific name.
		np := namePrinter{
			name: name,
		}
		go func() {
			// Submit the task to be worked on. When RunTask
			// returns we know it is being handled.
			if err := p.Run(&np); err != nil {
				spew.Dump(err)
			}
			wg.Done()
		}()

	}
	wg.Wait()
	// Shutdown the work pool and wait for all existing work
	// to be completed.
	p.Shutdown()
}

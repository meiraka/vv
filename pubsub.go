package main

import (
	"fmt"
	"sync"
)

/*PubSub is simple string Publish-Subscribe model.*/
type PubSub struct {
	m               sync.Mutex
	subscribeChan   chan chan string
	unsubscribeChan chan chan string
	countChan       chan chan int
	notifyChan      chan pubSubNotify
	stopChan        chan struct{}
}

type pubSubNotify struct {
	message string
	errChan chan error
}

/*EnsureStart starts PubSub daemon if not started.*/
func (p *PubSub) EnsureStart() {
	p.m.Lock()
	defer p.m.Unlock()
	if p.subscribeChan == nil {
		p.subscribeChan = make(chan chan string)
		p.unsubscribeChan = make(chan chan string)
		p.countChan = make(chan chan int)
		p.notifyChan = make(chan pubSubNotify)
		p.stopChan = make(chan struct{})
		go p.run()
	}
}

/*EnsureStop stops PubSub daemon.

Stopped PubSub instance can not restart again.
*/
func (p *PubSub) EnsureStop() {
	p.EnsureStart()
	p.stopChan <- struct{}{}
}

func (p *PubSub) run() {
	subscribers := []chan string{}
loop:
	for {
		select {
		case c := <-p.subscribeChan:
			subscribers = append(subscribers, c)
		case c := <-p.unsubscribeChan:
			newSubscribers := []chan string{}
			for _, o := range subscribers {
				if o != c {
					newSubscribers = append(newSubscribers, o)
				}
			}
			subscribers = newSubscribers
		case pn := <-p.notifyChan:
			errcnt := 0
			for _, c := range subscribers {
				select {
				case c <- pn.message:
				default:
					errcnt++
				}
			}
			if errcnt > 0 {
				pn.errChan <- fmt.Errorf("failed to send %s notify, %d", pn.message, errcnt)
			} else {
				pn.errChan <- nil
			}
		case c := <-p.countChan:
			c <- len(subscribers)
		case <-p.stopChan:
			break loop
		}
	}
}

/*Subscribe adds new listener channel.*/
func (p *PubSub) Subscribe(c chan string) {
	p.EnsureStart()
	p.subscribeChan <- c
}

/*Unsubscribe removes exists listener channel.*/
func (p *PubSub) Unsubscribe(c chan string) {
	p.EnsureStart()
	p.unsubscribeChan <- c
}

/*Notify sends messanges to subscribed listeners.*/
func (p *PubSub) Notify(s string) error {
	p.EnsureStart()
	message := pubSubNotify{s, make(chan error)}
	p.notifyChan <- message
	return <-message.errChan
}

/*Count sends messanges to subscribed listeners.*/
func (p *PubSub) Count() int {
	p.EnsureStart()
	ci := make(chan int)
	p.countChan <- ci
	return <-ci
}

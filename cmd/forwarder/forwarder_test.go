package main

type nullForwarder struct{}

func (f *nullForwarder) Deliver(p payload) error {
	return nil
}

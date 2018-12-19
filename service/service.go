package service

import "github.com/go-ocf/kit/log"

func New() *Server {
	// CPU profiling by default
	//defer profile.Start().Stop()
	// Memory profiling
	//defer profile.Start(profile.MemProfile).Stop()

	//run server
	s, err := NewServer()
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Server config %v", *s)
	return s
}

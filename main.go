package main

func main() {
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

	err = s.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}

}

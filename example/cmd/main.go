package main

import (
	"fmt"
	"io/ioutil"
	"time"
	"watcher/pkg/watcher"
)

func main() {
	files := []string{"./f1", "./f2", "./f3"}
	s := watcher.Watch(time.Second*5, files)

	// for example write files async
	go func() {
		for _, f := range files {
			ioutil.WriteFile(f, []byte(f), 0644)
			time.Sleep(time.Second * 5)
		}
	}()

	for {
		changed := <-s.C
		fmt.Println(changed)
	}
}

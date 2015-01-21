package main

import (
	"fmt"
	"github.com/ddo/go-crawler"
)

func main() {
	no := 0

	c := crawler.New(&crawler.Config{
		Url: "http://facebook.com/",
	})

	receiver_url := func(url string) {
		no++
		fmt.Println(no, "\t ", url)
	}

	receiver_err := func(err error) {
		fmt.Println("error\t", err)
	}

	c.Start(receiver_url, receiver_err)

	fmt.Println("done thanks god")
}

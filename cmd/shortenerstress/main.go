package main

import (
	"flag"
	"fmt"
	"log"
	"net/http/cookiejar"
	"runtime"

	"github.com/go-resty/resty/v2"

	"github.com/Yandex-Practicum/go-autotests/internal/random"
)

var (
	flagGoroutinesNum uint64
	flagURLsNum       uint64
)

func main() {
	log.Println("Starting stress")

	flag.Uint64Var(&flagGoroutinesNum, "g", uint64(runtime.NumCPU()), "number of goroutines to use")
	flag.Uint64Var(&flagURLsNum, "n", 1000, "number of URLs to generate")
	flag.NArg()
	log.Println("Parsing flags")
	flag.Parse()

	if flag.NArg() != 2 {
		flag.Usage()
		log.Fatalln("Exactly one target shortener address must be specified after arguments")
	}

	address := flag.Arg(1)

	log.Printf("Running stress with following options: %v\n", map[string]interface{}{
		"address":          address,
		"goroutines_count": flagGoroutinesNum,
		"urls_count":       flagURLsNum,
	})

	if err := run(address); err != nil {
		log.Fatalf("unexpected error: %s", err)
	}

	log.Println("Stress ended successfully")
}

func run(target string) error {
	log.Println("Generating URLs")
	urls := make(chan string, int(flagURLsNum))
	for i := 0; i < int(flagURLsNum); i++ {
		urls <- random.URL().String()
	}

	log.Println("Creating HTTP client")
	jar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("cannot create cookie jar")
	}
	httpc := resty.New().
		SetCookieJar(jar).
		SetHostURL(target)

	log.Println("Performing requests")
	for i := 0; i < int(flagGoroutinesNum); i++ {
		go func() {
			for url := range urls {
				resp, err := httpc.R().Post("/")
				if err != nil {
					log.Printf("cannot perform request: %s\n", err)
					continue
				}
				log.Printf("got response %d for URL: %s\n", resp.StatusCode(), url)
			}
		}()
	}

	return nil
}

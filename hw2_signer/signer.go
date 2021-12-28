package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

func startJob(in, out chan interface{}, j job, wg *sync.WaitGroup) {
	j(in, out)
	close(out)
	wg.Done()
}

func ExecutePipeline(jobs ...job) {
	in := make(chan interface{})
	out := make(chan interface{})
	wg := &sync.WaitGroup{}

	for _, j := range jobs {
		wg.Add(1)
		go func(in, out chan interface{}, j job, wg *sync.WaitGroup) {
			j(in, out)
			close(out)
			wg.Done()
		}(in, out, j, wg)
		in = out
		out = make(chan interface{})
	}

	wg.Wait()
}

func DataSignerMd5_(data string, quotaCh chan struct{}, res chan string) {
	quotaCh <- struct{}{}
	res <- DataSignerMd5(data)
	<-quotaCh
}

func SingleHashWorker(data string, quotaCh chan struct{}, out chan interface{}, wg *sync.WaitGroup) {

	out1 := make(chan string)
	out2 := make(chan string)

	go func(value string, out chan string) {
		res := DataSignerCrc32(value)
		out <- res
	}(data, out1)

	go func(value string, out chan string) {
		md5ResChan := make(chan string)

		go DataSignerMd5_(value, quotaCh, md5ResChan)

		md5 := <-md5ResChan
		res := DataSignerCrc32(md5)
		out <- res
	}(data, out2)

	res1 := <-out1
	res2 := <-out2

	out <- res1 + "~" + res2
	wg.Done()
}

func SingleHash(in, out chan interface{}) {
	quotaCh := make(chan struct{}, 1)
	wg := &sync.WaitGroup{}

	for data := range in {
		wg.Add(1)
		go SingleHashWorker(strconv.Itoa(data.(int)), quotaCh, out, wg)
	}

	wg.Wait()
}

func MultiHashWorker(data string, out chan interface{}, wg *sync.WaitGroup) {
	var res string
	var results = map[int]string{}
	mu := &sync.Mutex{}
	wgWorker := &sync.WaitGroup{}

	wgWorker.Add(6)
	for i := 0; i < 6; i++ {
		go func(data string, i int, results map[int]string, wg *sync.WaitGroup, mu *sync.Mutex) {
			res := DataSignerCrc32(strconv.Itoa(i) + data)
			mu.Lock()
			results[i] = res
			mu.Unlock()
			wg.Done()
		}(data, i, results, wgWorker, mu)
	}
	wgWorker.Wait()

	for i := 0; i < 6; i++ {
		mu.Lock()
		res += results[i]
		mu.Unlock()
	}

	out <- res
	wg.Done()
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}

	for data := range in {
		wg.Add(1)
		go MultiHashWorker(data.(string), out, wg)
	}

	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	results := []string{}
	for data := range in {
		value := data.(string)
		results = append(results, value)
		sort.Strings(results)
	}

	out <- strings.Join(results, "_")
}

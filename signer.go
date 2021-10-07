package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	//"time"
)

func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{}
	in := make(chan interface{})
	for _, function := range jobs {
		out := make(chan interface{})
		wg.Add(1)
		go work(wg, function, in, out)
		in = out
	}
	wg.Wait()
	//	fmt.Println(<-in)
}

func work(wg *sync.WaitGroup, function job, in, out chan interface{}) {
	defer wg.Done()
	defer close(out)
	function(in, out)
}

func runCrc32(dataStr string, crc32res chan string) {
	var res string
	go func() {
		DataSignerCrc32(dataStr)
	}()
	crc32res <- res
}

func SingleHash(in, out chan interface{}) {
	for data := range in {
		fmt.Println(data)
		data_str := fmt.Sprintf("%d", data)
		fmt.Println(data_str, " SingleHash data ", data_str)
		res := "test"

		md5 := DataSignerMd5(data_str) //1 раз 10 мс

		crc32md5 := DataSignerCrc32(md5)
		crc32 := DataSignerCrc32(data_str)
		fmt.Println(data_str, "SingleHash crc32(md5(data))", crc32md5)
		fmt.Println(data_str, "SingleHash crc32(data)", crc32)

		res = crc32 + "~" + crc32md5
		fmt.Println(data_str, "0 SingleHash result", res)
		out <- res
	}
}

func runMultihash(data interface{}, out chan interface{}) {

	wg := &sync.WaitGroup{}
	var thResults = map[int]string{}
	mu := &sync.Mutex{}
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			th := fmt.Sprintf("%d", i)
			data_str := fmt.Sprintf("%s", data)
			iterCrc32 := DataSignerCrc32(th + data_str)
			mu.Lock()
			thResults[i] = iterCrc32
			mu.Unlock()
			fmt.Println(data, "MultiHash: crc32(th+step1))", th, iterCrc32)
			//res += iterCrc32 СЮДА НАДО СКЛАДЫВАТЬ!
		}(i)
	}

	wg.Wait()
	var joinedString string
	for i := 0; i < 6; i++ {
		joinedString += thResults[i]
	}

	out <- joinedString
}

func MultiHash(in, out chan interface{}) {
	start := time.Now()
	wgOut := &sync.WaitGroup{}
	for data := range in {
		wgOut.Add(1)
		fmt.Println("********************************************************", data)
		var res string
		go func() {
			wg := &sync.WaitGroup{}
			var thResults = map[int]string{}
			mu := &sync.Mutex{}
			for i := 0; i < 6; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					th := fmt.Sprintf("%d", i)
					data_str := fmt.Sprintf("%s", data)
					iterCrc32 := DataSignerCrc32(th + data_str)
					mu.Lock()
					thResults[i] = iterCrc32
					mu.Unlock()
					fmt.Println(data, "MultiHash: crc32(th+step1))", th, iterCrc32)
				}(i)
			}

			wg.Wait()

			var joinedString string
			for i := 0; i < 6; i++ {
				joinedString += thResults[i]
			}

			out <- joinedString
			wgOut.Done()
		}()

		wgOut.Wait()
		end := time.Since(start)
		fmt.Println(end, "???????????????????????????end")
		fmt.Println(data, "MultiHash result", res)

		//out <- res
	}
}

func _MultiHash(in, out chan interface{}) {

	for data := range in {
		var res string
		for i := 0; i < 6; i++ {

			th := fmt.Sprintf("%d", i)
			data_str := fmt.Sprintf("%s", data)

			iterCrc32 := DataSignerCrc32(th + data_str)
			fmt.Println(data, "MultiHash: crc32(th+step1))", th, iterCrc32)
			res += iterCrc32

		}
		fmt.Println(data, "MultiHash result", res)

		out <- res
	}
}

func CombineResults(in, out chan interface{}) {

	result := []string{}
	for x := range in {
		str := fmt.Sprintf("%v", x)
		time.Sleep(time.Second)
		result = append(result, str)
	}
	sort.Strings(result)

	joined_string := strings.Join(result, "_")
	fmt.Println(joined_string, "FINAL")
	out <- joined_string

}

func main() {
	testResult := "empty"
	//inputData := []int{0, 1, 1, 2, 3, 5, 8}
	//inputData := []int{0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 11, 1, 11, 1, 1, 11, 1, 1, 1}
	inputData := []int{0, 1}
	hashSignJobs := []job{
		job(func(in, out chan interface{}) {
			for _, fibNum := range inputData {
				out <- fibNum
			}
		}),
		//job(SingleHash),
		job(MultiHash),
		job(CombineResults),
		job(func(in, out chan interface{}) {
			dataRaw := <-in
			data, ok := dataRaw.(string)
			if !ok {
				fmt.Println("cant convert result data to string")
			}
			testResult = data
		}),
	}

	ExecutePipeline(hashSignJobs...)
	fmt.Println(testResult, "res")
	exeptedResult := "29568666068035183841425683795340791879727309630931025356555_4958044192186797981418233587017209679042592862002427381542"
	if testResult == exeptedResult {
		fmt.Println("!!!!!!!!!!!OKK")
	} else {
		fmt.Println("!!!!!!!!!!BAD")
	}
}

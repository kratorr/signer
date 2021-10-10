package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
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

}

func work(wg *sync.WaitGroup, function job, in, out chan interface{}) {
	defer wg.Done()
	defer close(out)
	function(in, out)
}

func SingleHash(in, out chan interface{}) {

	// 	* DataSignerMd5 может одновременно вызываться только 1 раз, считается 10 мс.
	// Если одновременно запустится несколько - будет перегрев на 1 сек
	// * DataSignerCrc32, считается 1 сек
	//  DataSignerCrc32(data)+"~"+DataSignerCrc32(DataSignerMd5(data))
	// crc32(data)+"~"+crc32(md5(data))
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	for data := range in {
		wg.Add(1)
		go func(data interface{}) {
			data_str := fmt.Sprintf("%d", data)
			fmt.Println(data_str, " SingleHash data ", data_str)
			time.Sleep(time.Millisecond * 10)
			mu.Lock()
			md5 := DataSignerMd5(data_str) //считаем что тут ок, типа 10 мс считаем и спим до и после.
			mu.Unlock()
			time.Sleep(time.Millisecond * 10)
			chanResStr := make(chan string)
			chanResMd := make(chan string)

			go func(data_str string) {
				crc32 := DataSignerCrc32(data_str)
				chanResStr <- crc32
			}(data_str)

			go func(md5 string) {
				crc32Md5 := DataSignerCrc32(md5)
				chanResMd <- crc32Md5
			}(md5)

			var crc32FromChan, crc32md5FromChan string
			go func() {
				for i := 0; i < 2; i++ {
					select {
					case mdRes := <-chanResMd:

						crc32md5FromChan = mdRes
					case crc32Res := <-chanResStr:

						crc32FromChan = crc32Res
					}
				}
				fmt.Println(data_str, "SingleHash crc32(md5(data))", crc32FromChan)
				fmt.Println(data_str, "SingleHash crc32(data)", crc32md5FromChan)
				res := crc32FromChan + "~" + crc32md5FromChan
				fmt.Println(data_str, "0 SingleHash result", res)
				out <- res
				wg.Done()
			}()

		}(data)

	}
	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	wgOut := &sync.WaitGroup{}
	for data := range in {
		wgOut.Add(1)

		go func(data interface{}) {
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
		}(data)

	}
	wgOut.Wait()

}

func CombineResults(in, out chan interface{}) {

	result := []string{}
	for data := range in {

		str := fmt.Sprintf("%v", data)
		result = append(result, str)
	}
	sort.Strings(result)

	joined_string := strings.Join(result, "_")

	out <- joined_string

}

func main() {
	testResult := "empty"
	//inputData := []int{0, 1, 1, 2, 3, 5, 8}
	inputData := []int{0, 1}
	hashSignJobs := []job{
		job(func(in, out chan interface{}) {
			for _, fibNum := range inputData {
				out <- fibNum
			}
		}),
		job(SingleHash),
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

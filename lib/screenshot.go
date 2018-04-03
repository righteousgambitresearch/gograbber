package lib

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/benbjohnson/phantomjs"
)

func Screenshot(s *State) (h []Host) {
	// for true {
	// 	page, err := s.PhantomProcesses.CreateWebPage()
	// 	if err != nil {
	// 		fmt.Println(err)
	// 		time.Sleep(time.Second)
	// 		page.Close()
	// 		continue
	// 	}
	// 	if err := page.Open("http://localhost:20202/"); err != nil {
	// 		fmt.Println(err)
	// 		time.Sleep(time.Second)
	// 		page.Close()
	// 		continue
	// 	}
	// 	page.Close()
	// 	break
	// }
	hostChan := make(chan Host, s.Threads)

	wg := sync.WaitGroup{}
	targetHost := make(TargetHost, s.Threads)
	var cnt int
	for _, host := range s.URLComponents {
		// wg.Add(1)
		// go distributeScreenshotWorkers(s, URLComponent, hostChan, respChan, &wg)
		for path := range host.Paths.Set {
			wg.Add(1) //MAKE SURE SCREENSHOTURL HAS A DONE CALL IN IT JFC
			routineId := Counter{cnt}
			targetHost <- routineId
			go targetHost.ScreenshotAURL(s, cnt, host, path, hostChan, &wg)
			cnt++
		}
	}

	go func() {
		for url := range hostChan {
			h = append(h, url)
		}
	}()
	wg.Wait()
	close(hostChan)
	// write resps to file? return hosts for now
	return h
}

// func distributeScreenshotWorkers(s *State, host Host, hostChan chan Host, respChan chan *http.Response, wg *sync.WaitGroup) {
// 	//wg.Add called before this, so we FUCKING DEFER DONE IT
// 	defer wg.Done()
// 	for path := range host.Paths.Set {
// 		wg.Add(1) //MAKE SURE SCREENSHOTURL HAS A DONE CALL IN IT JFC
// 		go ScreenshotAURL(s, host, path, hostChan, respChan, wg)
// 	}
// }

func (target TargetHost) ScreenshotAURL(s *State, cnt int, host Host, path string, hostChan chan Host, wg *sync.WaitGroup) (err error) {
	defer wg.Done()
	page, err := s.PhantomProcesses[cnt%len(s.PhantomProcesses)].CreateWebPage()
	page.SetSettings(phantomjs.WebPageSettings{ResourceTimeout: s.Timeout}) // Time out the page if it takes too long to load. Sometimes JS is fucky and takes wicked long to do nothing forever :(
	url := fmt.Sprintf("%v://%v:%v/%v", host.Protocol, host.HostAddr, host.Port, path)

	if err != nil {
		fmt.Printf("Unable to Create webpage: %v (%v)\n", url, err)
		<-target

		return err
	}
	defer page.Close()

	if strings.HasPrefix(path, "/") {
		path = path[1:] // strip preceding '/' char
	}
	if s.Debug {
		fmt.Printf("Trying to screenshot URL: %v\n", url)
	}
	if s.Jitter > 0 {
		jitter := time.Duration(rand.Intn(s.Jitter)) * time.Millisecond
		if s.Debug {
			fmt.Printf("Jitter: %v\n", jitter)
		}
		time.Sleep(jitter)
	}
	if err := page.Open(url); err != nil {
		fmt.Printf("Unable to open page: %v (%v)\n", url, err)
		<-target

		return err
	}
	// Setup the viewport and render the results view.
	if err := page.SetViewportSize(s.ImgX, s.ImgY); err != nil {
		fmt.Printf("Unable to set Viewport size: %v (%v)\n", url, err)
		<-target

		return err
	}
	currTime := strings.Replace(time.Now().Format(time.RFC3339), ":", "_", -1)
	var screenshotFilename string
	if s.ProjectName != "" {
		screenshotFilename = fmt.Sprintf("%v/%v_%v_%v_%v-%v_%v.png", s.ScreenshotDirectory, strings.ToLower(strings.Replace(s.ProjectName, " ", "_", -1)), host.Protocol, host.HostAddr, host.Port, currTime, rand.Int63())
	} else {
		screenshotFilename = fmt.Sprintf("%v/%v_%v_%v-%v_%v.png", s.ScreenshotDirectory, host.Protocol, host.HostAddr, host.Port, currTime, rand.Int63())
	}
	fmt.Println(screenshotFilename)
	if err := page.Render(screenshotFilename, "png", s.ScreenshotQuality); err != nil {
		fmt.Printf("Unable to save Screenshot: %v (%v)\n", url, err)
		<-target

		return err
	}
	host.ScreenshotFilename = screenshotFilename
	hostChan <- host
	<-target
	return
}

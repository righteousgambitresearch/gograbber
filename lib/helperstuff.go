package lib

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Shim type for "set" containing ints
type IntSet struct {
	Set map[int]bool
}

// Shim type for "set" containing strings
type StringSet struct {
	Set map[string]bool
}

type Host struct {
	Paths    StringSet
	HostAddr string
	Port     int
	Protocol string
}

var tx = &http.Transport{
	DialContext: (&net.Dialer{
		//transports don't have default timeouts because having sensible defaults would be too good
		Timeout: 5 * time.Second,
	}).DialContext,
	TLSHandshakeTimeout:   10 * time.Second,
	MaxIdleConns:          100, //This could potentially be dropped to 1, we aren't going to hit the same server more than once ever
	IdleConnTimeout:       5 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	DisableKeepAlives:     true,
	DisableCompression:    true,
	TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
}

var cl = http.Client{
	Transport: tx,
	Timeout:   time.Second * 5, //eyy no reasonable timeout on clients too!
}

func Hosts(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}
	return ips, nil
}

//  http://play.golang.org/p/m8TNTtygK0
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// ExpandHosts takes a string array of IP addresses/CIDR masks and converts into a string array of pure IP addresses
func ExpandHosts(targets []string, ch chan StringSet) {
	allHosts := StringSet{Set: map[string]bool{}} // Initialise the hosts list... nfi why this is a thing?
	for _, target := range targets {
		ips, err := Hosts(target)
		if err != nil { // Not a CIDR... Might be a straight IP or url
			ip := net.ParseIP(target)
			if ip == nil {
				continue
			}
			allHosts.Add(ip.String())
		}
		allHosts.AddRange(ips)
	}
	ch <- allHosts
}

// LeftPad2Len https://github.com/DaddyOh/golang-samples/blob/master/pad.go
func LeftPad2Len(s string, padStr string, overallLen int) string {
	var padCountInt int
	padCountInt = 1 + ((overallLen - len(padStr)) / len(padStr))
	var retStr = strings.Repeat(padStr, padCountInt) + s
	return retStr[(len(retStr) - overallLen):]
}

// RightPad2Len https://github.com/DaddyOh/golang-samples/blob/master/pad.go
func RightPad2Len(s string, padStr string, overallLen int) string {
	var padCountInt int
	padCountInt = 1 + ((overallLen - len(padStr)) / len(padStr))
	var retStr = s + strings.Repeat(padStr, padCountInt)
	return retStr[:overallLen]
}

// readLines reads a whole file into memory
// and returns a slice of its lines.
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func GetDataFromFile(fileName string) (data []string, err error) {
	if fileName != "" {
		data, err := readLines(fileName)
		if err != nil {
			fmt.Printf("File: %v does not exist, or you do not have permz (%v)\n", fileName, err)
			return nil, err
		}
		return data, err
	}
	return
}

// Taken from gobuster - THANKS! /**/
// StrArrToInt takes an array of strings and *hopefully* returns an array of ints?
func StrArrToInt(t []string) (t2 []int) {
	for _, i := range t {
		j, err := strconv.Atoi(i)
		if err != nil {
			panic(err)
		}
		t2 = append(t2, j)
	}
	return t2
}

// Add an element to a set
func (set *StringSet) Add(s string) bool {
	_, found := set.Set[s]
	set.Set[s] = true
	return !found
}

// Add a list of elements to a set
func (set *StringSet) AddRange(ss []string) {
	for _, s := range ss {
		set.Set[s] = true
	}
}

// Test if an element is in a set
func (set *StringSet) Contains(s string) bool {
	_, found := set.Set[s]
	return found
}

// Check if any of the elements exist
func (set *StringSet) ContainsAny(ss []string) bool {
	for _, s := range ss {
		if set.Set[s] {
			return true
		}
	}
	return false
}

// Stringify the set
func (set *StringSet) Stringify() string {
	values := []string{}
	for s, _ := range set.Set {
		values = append(values, s)
	}
	return strings.Join(values, ",")
}

// Add an element to a set
func (set *IntSet) Add(i int) bool {
	_, found := set.Set[i]
	set.Set[i] = true
	return !found
}

// Test if an element is in a set
func (set *IntSet) Contains(i int) bool {
	_, found := set.Set[i]
	return found
}

// Stringify the set
func (set *IntSet) Stringify() string {
	values := []string{}
	for s, _ := range set.Set {
		values = append(values, strconv.Itoa(s))
	}
	return strings.Join(values, ",")
}

/**/
func ChunkString(s string, chunkSize int) []string {
	var chunks []string
	runes := []rune(s)

	if len(runes) == 0 {
		return []string{s}
	}

	for i := 0; i < len(runes); i += chunkSize {
		nn := i + chunkSize
		if nn > len(runes) {
			nn = len(runes)
		}
		chunks = append(chunks, string(runes[i:nn]))
	}
	return chunks
}

func GenerateURLs(targetList StringSet, Protocol string, Ports IntSet, Paths StringSet, ch chan []Host) {
	var HostStructs []Host
	for target, _ := range targetList.Set {
		for port, _ := range Ports.Set {
			HostStructs = append(HostStructs, Host{Port: port, Protocol: Protocol, HostAddr: target, Paths: Paths})
		}
	}
	ch <- HostStructs
}
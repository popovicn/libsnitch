package main

import (
    "bufio"
    "encoding/json"
    "flag"
    "fmt"
    "math/rand"
    "net/http"
    "net/url"
    "os"
    "strings"
    "sync"
    "time"
)

var banner = `
    _    _ ___ ` + BannerColor + ` ____ _  _ _ ___ ____ _  _ ` + RstColor + `
    |    | |__]` + BannerColor + ` l__  |\ | |  |  |    |__| ` + RstColor + `
    l___ | |__]` + BannerColor + ` ___] | \| |  |  l___ |  | ` + RstColor + `


`+ RstColor

const (
    Red = "\033[31;1m"
    Green = "\033[32;1m"
    RstColor = "\033[0m"
    BannerColor = Red
)

// Dependency Manager

type DependencyManager struct {
    techName string
    mu sync.Mutex
    dependencyLocks map[string]*sync.Mutex
    dependencyData map[string]int
    totalDependencies int
    brokenDependencies int
}

func InitDependencyManager(techName string) *DependencyManager {
    return &DependencyManager{
        techName: techName,
        dependencyLocks: make(map[string]*sync.Mutex),
        dependencyData: make(map[string]int),
        totalDependencies: 0,
        brokenDependencies: 0,
    }
}

func (dm *DependencyManager) GetMutex(packageName string) *sync.Mutex {
    dm.mu.Lock()
    defer dm.mu.Unlock()
    if mutex, found := dm.dependencyLocks[packageName]; found {
        return mutex
    } else {
        dm.dependencyLocks[packageName] = &sync.Mutex{}
        return dm.dependencyLocks[packageName]
    }
}

func (dm *DependencyManager) GetPackageInfo(packageName string) (int, bool) {
    dm.mu.Lock()
    defer dm.mu.Unlock()
    status, found := dm.dependencyData[packageName]
    return status, found
}

func (dm *DependencyManager) SetPackageInfo(packageName string, status int) {
    dm.mu.Lock()
    defer dm.mu.Unlock()
    dm.totalDependencies += 1
    if status != 200 {
        dm.brokenDependencies += 1
    }
    dm.dependencyData[packageName] = status
}

// Statistics

type Stats struct {
    mu sync.Mutex
    totalTargets      int
    packageJsonParsed int
}

func InitStats() *Stats {
    return &Stats {
        totalTargets: 0,
        packageJsonParsed: 0,
    }
}

func (s *Stats) IncTotalTargets() {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.totalTargets += 1
}

func (s *Stats) IncPackageJsonParsed() {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.packageJsonParsed += 1
}

// Variables

var userAgents = []string {
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.182 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 11_2_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.182 Safari/537.36",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.182 Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:85.0) Gecko/20100101 Firefox/85.0",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 11.2; rv:85.0) Gecko/20100101 Firefox/85.0",
    "Mozilla/5.0 (X11; Linux i686; rv:85.0) Gecko/20100101 Firefox/85.0",
    "Mozilla/5.0 (Android 11; Mobile; rv:68.0) Gecko/68.0 Firefox/85.0",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/87.0.4280.77 Mobile/15E148 Safari/604.1",
    "Mozilla/5.0 (Linux; Android 10) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.181 Mobile Safari/537.36",
}

var stats *Stats
var nodeManager *DependencyManager
var mu sync.Mutex
var wg sync.WaitGroup
var parallelism *int
var npmDelay *float64
var timeout time.Duration
var simpleCli *bool
var outputFilePath *string
var targetDomains [] string
var activeThreads chan interface{}
var npmMutex sync.Mutex
var outputMutex sync.Mutex

func printStats(elapsedTime time.Duration) {
    fmt.Println()
    fmt.Printf("‣ Succeeded in %s \n", elapsedTime.String())
    fmt.Printf("‣    targets scanned          %d\n", stats.totalTargets)
    fmt.Printf("‣    exposed package.json     %d\n", stats.packageJsonParsed)
    fmt.Printf("‣    tested npm dependencies  %d\n", nodeManager.totalDependencies)
    if nodeManager.brokenDependencies == 1 {
        fmt.Printf("‣ Found %d broken dependency. \n", nodeManager.brokenDependencies)
    } else {
        fmt.Printf("‣ Found %d broken dependencies. \n", nodeManager.brokenDependencies)
    }
    fmt.Println(RstColor)
}

func writeResult(line, outputFile string) {
    outputMutex.Lock()
    defer outputMutex.Unlock()
    file, _ := os.OpenFile(outputFile, os.O_APPEND|os.O_WRONLY, 0600)
    defer file.Close()
    writer := bufio.NewWriter(file)
    _, _ = fmt.Fprintln(writer, line)
    _ = writer.Flush()
}

func handleResult(targetUrl, packageName, dependencyType string, status int){
    mu.Lock()
    defer mu.Unlock()
    parsedUrl, _ := url.Parse(targetUrl)
    resultLine := fmt.Sprintf("%d\t%s\t%s\t%s", status, targetUrl, packageName, dependencyType)
    if *simpleCli {
        fmt.Println(resultLine)
    } else {
        if status == 200 {
            fmt.Printf("%s%d%s\t%s  \t%s (%s)\n", Green, status, RstColor, parsedUrl.Host, packageName, dependencyType)
        } else {
            fmt.Printf("%s%d\t%s  \t%s (%s)%s\n", Red, status, parsedUrl.Host, packageName, dependencyType, RstColor)
        }
    }
    if *outputFilePath != "" {
        writeResult(resultLine, *outputFilePath)
    }

}

func handleError(url, msg string) {
    mu.Lock()
    defer mu.Unlock()
    fmt.Println(url, Red, "Error", msg, RstColor)
}

func testNpmDependency(targetUrl, packageName, dependencyType string) {
    nodeManager.GetMutex(packageName).Lock()
    defer nodeManager.GetMutex(packageName).Unlock()
    if statusCode, found := nodeManager.GetPackageInfo(packageName); found {
        handleResult(targetUrl, packageName, dependencyType, statusCode)
        return
    }
    activeThreads <- struct{}{}
    defer func(){ <- activeThreads }()

    // Delay requests
    if *npmDelay > 0.0 {
        npmMutex.Lock()
        defer npmMutex.Unlock()
        time.Sleep(time.Duration(*npmDelay*1000) * time.Millisecond)
    }

    request, reqErr := http.NewRequest("GET", "https://www.npmjs.com/package/" + packageName, nil)
    if reqErr != nil {
        return
    }
    request.Header.Set("User-Agent", getRandomUserAgent())
    client := &http.Client{Timeout: timeout}
    response, respError := client.Do(request)
    if respError != nil || response == nil {
        return
    }
    nodeManager.SetPackageInfo(packageName, response.StatusCode)
    handleResult(targetUrl, packageName, dependencyType, response.StatusCode)
}

func snitchNodeJs(targetUrl string) {
    activeThreads <- struct{}{}
    defer func(){ <- activeThreads }()
    defer wg.Done()

    stats.IncTotalTargets()
    u, _ := url.Parse(targetUrl)
    targetUrl = fmt.Sprintf("%s://%s/package.json", u.Scheme, u.Host)
    client := &http.Client{
        Timeout: timeout,
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
           return http.ErrUseLastResponse
        },
    }
    request, reqErr := http.NewRequest("GET", targetUrl, nil)
    if reqErr != nil {
        handleError(targetUrl, reqErr.Error())
        return
    }
    request.Header.Set("User-Agent", getRandomUserAgent())
    response, err := client.Do(request)
    if err != nil || response == nil || response.StatusCode != 200 ||
        !strings.Contains(response.Header.Get("Content-Type"), "application/json") {
        return
    }
    defer response.Body.Close()
    packageJson := make(map[string]interface{})
    decodeErr := json.NewDecoder(response.Body).Decode(&packageJson)
    if decodeErr != nil {
        handleError(targetUrl, "c")
        return
    }
    stats.IncPackageJsonParsed()
    for key := range packageJson {
        if strings.Contains(strings.ToLower(key), "dependencies") {
            dependencyMap, ok := packageJson[key].(map[string]interface{})
            if ok {
                for packageName := range dependencyMap {
                    testNpmDependency(targetUrl, packageName, key)
                }
            }
        }
    }
}

func runLibSnitch() {
    for _, domain := range targetDomains {
        wg.Add(1)
        go snitchNodeJs(domain)
    }
    wg.Wait()
}

func abort(msg string, code int) {
    fmt.Println("Error:", msg)
    os.Exit(code)
}

func getRandomUserAgent() string {
    rand.Seed(time.Now().UnixNano())
    return userAgents[rand.Intn(len(userAgents))]
}

func readInputFile(path string) []string {
    file, err := os.Open(path)
    if err != nil {
        abort(err.Error(), 1)
    }
    defer file.Close()

    var lines []string
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }
    if scanner.Err() != nil { abort(scanner.Err().Error(), 1)}
    return lines
}

func main(){
    var targetDomain = flag.String("d", "", "Target domain")
    var inputFilePath = flag.String("df", "","Input file path")
    parallelism = flag.Int("p", 50, "Parallelism")
    argTimeout := flag.Int("t", 10, "Request timeout in seconds.")
    npmDelay = flag.Float64("npmd", 0.0, "Delay seconds between requests to npmjs.com")
    simpleCli = flag.Bool("s", false, "Simple CLI (default: false)")
    outputFilePath = flag.String("o", "","Output file path")
    flag.Parse()
    timeout = time.Duration(*argTimeout) * time.Second
    activeThreads = make(chan interface{}, *parallelism)
    stats = InitStats()
    nodeManager = InitDependencyManager("Node")

    // Prepare target domain list
    if *targetDomain != "" {
        targetDomains = []string{*targetDomain}
    } else {
        if *inputFilePath == "" {
            fmt.Println()
            flag.PrintDefaults()
            fmt.Println()
            abort("Must specify target domain or input file path.", 1)
        }
        targetDomains = readInputFile(*inputFilePath)
    }
    // Try creating output file if provided path
    if *outputFilePath != "" {
        _, err := os.Create(*outputFilePath)
        if err != nil {
            abort(err.Error(), 1)
        }
    }

    if !*simpleCli {
        fmt.Print(banner)
    }

    start := time.Now()
    runLibSnitch()
    elapsed := time.Since(start)

    if !*simpleCli {
        printStats(elapsed)
    }
}
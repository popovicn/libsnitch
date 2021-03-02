libsnitch
---------
Find broken npm dependencies from exposed package.json

### Example usage
```
➜ go run libsnitch.go -df domains-500.txt -p 100 -npmd 0.4 -o output.txt 

    _    _ ___  ____ _  _ _ ___ ____ _  _ 
    |    | |__] l__  |\ | |  |  |    |__| 
    l___ | |__] ___] | \| |  |  l___ |  | 


200	localhost:5000  	grunt (dependencies)
200	www.managingmadrid.com  	grunt-contrib-watch (devDependencies)
200	www.seqwater.com.au  	cypress (devDependencies)
...
404	localhost:5000  	dead-dependency-123 (devDependencies)
200	www.ridiculousupside.com  	grunt-stripmq (devDependencies)


‣ Succeeded in 16.380843107s 
‣    targets scanned          500
‣    exposed package.json     4
‣    tested npm dependencies  14
‣ Found 1 broken dependency. 
```
**output.txt**
```
200	http://localhost:5000/package.json	grunt	dependencies
200	http://localhost:5000/package.json	grunt-contrib-watch	devDependencies
200	https://www.ridiculousupside.com/package.json	grunt-newer	devDependencies
200	https://www.managingmadrid.com/package.json	grunt-contrib-jshint	devDependencies
200	https://www.managingmadrid.com/package.json	grunt-contrib-watch	devDependencies
...
```

### Arguments

| arg | type | Description |
| --- | ---- | ----------- |
| `-d` | string | Target domain |
| `-df` | string | Input file path | 
| `-p` | int | Parallelism (default 50)| 
| `-npmd` | float | Delay seconds between requests to [npmjs.com](https://www.npmjs.com/) (default 0) | 
| `-t` | int | Request timeout in seconds (default 10) | 
| `-s` | bool | Simple CLI | 
| `-o` | string | Output file path | 

/* linesrv.go - write lines to a web client one by one
 */

package main

import (
	"net/http"
	"io"
	"fmt"
	"log"
	"os/exec"
	"bufio"
	"runtime"
)

// First some static HTML
var head = `<html>
<head>
<title>Line Flushing Server</title>
<script type="text/javascript" src="https://ajax.googleapis.com/ajax/libs/jquery/1.8.2/jquery.min.js"></script>
<script type="text/javascript">

var stop = false, stopped = 0,
scrolldown = function() {
	if (stop && stopped++ > 10) {
		return;
	}
	$("html, body").animate({
		scrollTop: $(document).height()
	}, 10);
}
window.setInterval(scrolldown , 100);
$(document).ready(function() {
	stop = true;
});
</script>
</head>
<body>
<ul>
<li><a href="http://golang.org/">Go language homepage</a></li>
</ul>
<pre>
`
var foot = `
</pre>
</body>
</html>
`

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, head)
	c := make(chan string)
	go shellcmd(c)
	for s := range c {
		fmt.Fprintln(w, s)
		w.(http.Flusher).Flush()
	}
	fmt.Fprint(w, foot)
}

func shellcmd(sink chan string) {
	c := exec.Command("sh", "-c", "i=1; while test $i -lt 15; do echo $i; sleep 1; i=`expr $i + 1`; done")
	fromcmd, err := c.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	fromcmdb := bufio.NewReader(fromcmd)

	log.Println("starting")
	if err := c.Start(); err != nil {
		log.Fatal(err)
	}

	var line []byte
	for err == nil {
		line, _, err = fromcmdb.ReadLine()
		if err == nil || err == io.EOF {
			sink <- string(line)
		}
	}
	if err != io.EOF {
		log.Fatal(err)
	}
	if err := c.Wait(); err != nil {
		log.Print(err)
	}
	close(sink)
}

func main() {
	n := runtime.NumCPU()
	runtime.GOMAXPROCS(n)
	log.Printf("set GOMAXPROCS to %d\n", n)
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

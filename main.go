package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	miner string
	gae   string
	timer int
	id    string
}

type MinerData struct {
	Id   string
	Data string
}

var config Config

func readconfig(configfile string) {
	log.Printf("open config file: %s\n", configfile)
	file, err := os.Open(configfile)
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line[0] != '#' {
			params := strings.Split(line, "=")
			if len(params) == 2 {
				if params[0] == "miner" {
					config.miner = params[1]
				} else if params[0] == "gae" {
					config.gae = params[1]
				} else if params[0] == "id" {
					config.id = params[1]
				} else if params[0] == "timer" {
					timer, _ := strconv.ParseInt(params[1], 10, 32)
					config.timer = int(timer)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func writeconfig(configfile string) {
	file, err := os.OpenFile(configfile, os.O_CREATE, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	if _, err = file.WriteString(fmt.Sprintf("id=%s", config.id)); err != nil {
		log.Fatal(err)
	}
}

func readurl(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
	} else {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		} else {
			return string(body)
		}
	}
	return ""
}

func postdata(data string) string {
	var md MinerData
	md.Id = config.id
	md.Data = data
	js, _ := json.Marshal(&md)
	//jsb := byte[](js)
	req, err := http.NewRequest("POST", config.gae, bytes.NewBuffer(js))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return "error"
	}
	defer resp.Body.Close()
	return "ok"
}

func readminerjson() string {
	minerhtml := readurl(config.miner)
	if len(minerhtml) == 0 {
		return ""
	}
	data := strings.Split(minerhtml, "\n")
	line := "nil"
	for i := 0; i < 5; i++ {
		if len(data[i]) > 0 {
			if data[i][0] == '{' {
				line = data[i]
			}
		}
	}
	return line
}

func timerproc() {
	log.Println("timerproc()")
	minerjson := readminerjson()
	if minerjson == "" {
		log.Println("miner not found")
	} else {
		log.Println(minerjson)
	}
	log.Println("postdata()")
	postdata(minerjson)

}

func getcrc32(q string) string {
	crc32q := crc32.MakeTable(0xD5828281)
	ret := fmt.Sprintf("%08x", crc32.Checksum([]byte(q), crc32q))
	return ret
}

func genid() string {
	t := time.Now()
	st := t.Format(time.RFC1123)
	return getcrc32(st)
}

func main() {
	log.Println("minermon 0.5go")
	if len(os.Args) == 2 {
		configfile := os.Args[1]
		readconfig(configfile)
		readconfig("id.config")
		if config.miner != "" {
			log.Printf("miner: %s\n", config.miner)
			log.Printf("gae: %s\n", config.gae)
			log.Printf("timer: %s\n", config.timer)

			if config.id == "" {
				config.id = genid()
				writeconfig("id.config")
			}

			log.Printf("id: %s\n", config.id)
			for {
				timerproc()
				timer := time.NewTimer(time.Second * time.Duration(config.timer))
				<-timer.C
			}
		} else {
			log.Println("wrong config file!")
		}
	} else {
		log.Println("Usage: minermon <file.config>")
	}
}

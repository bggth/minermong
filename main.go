package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	mode  string
	miner string
	psw   string
	gae   string
	timer int
	id    string
}

type MinerData struct {
	Id   string
	Data string
}

type ClaymoreMinerData struct {
	Result [9]string `json:"result"`
}

type ClaymoreMinerRequest struct {
	ID      int    `json:"id"`
	Psw     string `json:"psw"`
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
}

type ClaymoreMinerResponce struct {
	ID     int       `json:"id"`
	Result [9]string `json:"result"`
}

type EWBFMinerRequest struct {
	ID     uint   `json:"id"`
	Method string `json:"method"`
}

type EWBFMinerResponce struct {
	ID     uint            `json:"id"`
	Method string          `json:"method"`
	Error  string          `json:"error"`
	Server string          `json:"current_server"`
	Result []EWBFMinerData `json:"result"`
}

type EWBFMinerData struct {
	Gpuid          uint   `json:"gpuid"`
	Cudaid         uint   `json:"cudaid"`
	Busid          string `json:"busid"`
	GpuStatus      uint   `json:"gpu_status"`
	Solver         uint   `json:"solver"`
	Temperature    int    `json:"temperature"`
	GpuPowerUsage  uint   `json:"gpu_power_usage"`
	SpeedSps       uint   `json:"speed_sps"`
	AcceptedShares uint   `json:"accepted_shares"`
	RejectedShares uint   `json:"rejected_shares"`
}

var config Config
var startTime time.Time

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
		if len(line) > 0 {
			if line[0] != '#' {
				params := strings.Split(line, "=")
				if len(params) == 2 {
					if params[0] == "miner" {
						config.miner = params[1]
					} else if params[0] == "gae" {
						config.gae = params[1]
					} else if params[0] == "id" {
						config.id = params[1]
					} else if params[0] == "psw" {
						config.psw = params[1]
					} else if params[0] == "timer" {
						timer, _ := strconv.ParseInt(params[1], 10, 32)
						config.timer = int(timer)
					} else if params[0] == "mode" {
						config.mode = params[1]
					}
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

func readclaymore() string {
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

//----------------------------------------------------------------------------------
//----------------------------------------------------------------------------------
//----------------------------------------------------------------------------------
//----------------------------------------------------------------------------------
//----------------------------------------------------------------------------------
func readewbf() string {

	log.Println("readewbf()")

	req := EWBFMinerRequest{
		ID:     1,
		Method: "getstat",
	}

	req_json, _ := json.Marshal(&req)
	log.Println("send", string(req_json))

	conn, err := net.Dial("tcp", config.miner)
	if err != nil {
		return "err"
	}
	fmt.Fprintf(conn, string(req_json)+"\n")
	resp_json, _ := bufio.NewReader(conn).ReadString('\n')
	log.Println("recv", resp_json)
	conn.Close()
	resp := EWBFMinerResponce{}
	json.Unmarshal([]byte(resp_json), &resp)
	return ewbf2claymore(resp)
}

func readclaymore2() string {
	log.Println("readclaymore2()")
	req := ClaymoreMinerRequest{
		ID:      1,
		Psw:     config.psw,
		Jsonrpc: "2.0",
		Method:  "miner_getstat1",
	}

	req_json, _ := json.Marshal(&req)
	log.Println("send", string(req_json))

	conn, err := net.Dial("tcp", config.miner)
	if err != nil {
		return "err"
	}
	fmt.Fprintf(conn, string(req_json)+"\n")
	resp_json, _ := bufio.NewReader(conn).ReadString('\n')
	log.Println("recv", resp_json)
	conn.Close()
	resp := ClaymoreMinerResponce{}
	//log.Println("resp_json", string(resp_json))
	json.Unmarshal([]byte(resp_json), &resp)
	log.Println(resp.ID, resp.Result)
	ret := ClaymoreMinerData{Result: resp.Result}
	ret_json, _ := json.Marshal(&ret)
	//log.Println("ret", string(ret_json))
	return string(ret_json)
}

func inittime() {
	startTime = time.Now()
}

func uptime() string {
	delta := time.Now().Sub(startTime)
	//log.Println("%s | %s", time.Now(), startTime)
	return fmt.Sprintf("%0.f", delta.Minutes())
}

func ewbf2claymore(ewbf EWBFMinerResponce) string {

	count := len(ewbf.Result)
	log.Printf("len(ewbf)=%d", count)
	cd := ClaymoreMinerData{}
	cd.Result[0] = "EWBF - ZEC"
	cd.Result[1] = uptime() //
	var hashrate int32
	var accepted int32
	var rejected int32
	var hashrates string
	var temps string
	for i := 0; i < count; i++ {
		hashrate = hashrate + 1000*int32(ewbf.Result[i].SpeedSps)
		hashrates = hashrates + fmt.Sprintf("%d;", 1000*ewbf.Result[i].SpeedSps)
		temps = temps + fmt.Sprintf("%d;0;", ewbf.Result[i].Temperature)
		accepted = accepted + int32(ewbf.Result[i].AcceptedShares)
		rejected = rejected + int32(ewbf.Result[i].RejectedShares)
	}
	cd.Result[2] = fmt.Sprintf("%d;%d;%d", hashrate, accepted, rejected) // hashrate; accepted; rejected
	cd.Result[3] = hashrates[0 : len(hashrates)-1]
	cd.Result[6] = temps[0 : len(temps)-1]
	if ewbf.Server != "" {
		cd.Result[7] = ewbf.Server
	} else {
		cd.Result[7] = "unknown"
		log.Println("Please update ewbf's miner for version 0.3.4b+")
	}

	cd.Result[8] = "0;0;0;0"
	cd_json, _ := json.Marshal(&cd)
	log.Println(string(cd_json))
	return string(cd_json)
}

func timerproc() {
	log.Println("timerproc()")

	if config.mode == "claymore" {
		minerjson := readclaymore2()
		if minerjson == "err" {
			log.Println("miner not found")
		} else {
			log.Println(minerjson)
			log.Println("postdata()")
			postdata(minerjson)
		}
		return
	}

	if config.mode == "ewbf" {
		data := readewbf()
		if data != "err" {
			log.Println("postdata()")
			postdata(data)
		} else {
			log.Println("miner not found")
		}
		return
	}

	log.Println("wrong mode in config file")

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

	inittime()

	log.Println("minermon 0.6go 09.09.2017")
	if len(os.Args) == 2 {
		configfile := os.Args[1]
		readconfig(configfile)
		if config.mode == "" {
			config.mode = "claymore" // claymore mode for default
		}
		readconfig("id.config")
		if config.miner != "" {
			log.Printf("mode: %s\n", config.mode)
			log.Printf("miner: %s\n", config.miner)
			log.Printf("psw: %s\n", config.psw)
			log.Printf("gae: %s\n", config.gae)
			log.Printf("timer: %d\n", config.timer)

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

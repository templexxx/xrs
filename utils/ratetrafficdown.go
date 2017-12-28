package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	var oneShardLost float64
	oneShardLost = 0.9808
	ds := os.Args[1]
	ps := os.Args[2]
	shardSizeS := os.Args[3]
	d, _ := strconv.ParseFloat(ds, 64)
	p, _ := strconv.ParseFloat(ps, 64)
	size, _ := strconv.ParseFloat(shardSizeS, 64)
	lostInData := d / (d + p) * oneShardLost
	rate := fmt.Sprintf("%.2f", lostInData*100)
	fmt.Println("chance of traffic down :", rate+`%`)
	lostOneDataCoff := (d + (d / (p - 1))) / (2 * d)
	avgT := lostInData*d*size*lostOneDataCoff + (1-lostInData)*d*size
	avgTraffic := fmt.Sprintf("%.2f", avgT)
	fmt.Println("avg repair traffic in xrs:", avgTraffic)
	rsTraffic := fmt.Sprintf("%.2f", d*size)
	fmt.Println("repair traffic in rs codes:", rsTraffic)
	trafficDown := fmt.Sprintf("%.2f", (d*size-avgT)/(d*size)*100)
	fmt.Println("rate of traffic down :", trafficDown+`%`)
}

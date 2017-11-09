package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/monotone/go-jd/core"
	clog "gopkg.in/clog.v1"
)

func init() {
	if err := clog.New(clog.CONSOLE, clog.ConsoleConfig{
		Level:      clog.TRACE,
		BufferSize: 100},
	); err != nil {
		fmt.Printf("init console log failed. error %+v.", err)
		os.Exit(1)
	}
}

const (
	AreaBeijing                      = "1_72_2799_0"
	Area_HuNan_ChangShaShi_KaiFuQu   = "18-1482-48938"
	AreaHunanShaoyangShaodongChengqu = "18_1511_1513_40429"
)

var (
	area   = flag.String("area", AreaHunanShaoyangShaodongChengqu, "ship location string, default to Beijing")
	period = flag.Int("period", 500, "the refresh period when out of stock, unit: ms.")
	rush   = flag.Bool("rush", false, "continue to refresh when out of stock.")
	order  = flag.Bool("order", false, "submit the order to JingDong when get the Goods.")
	goods  = flag.String("goods", "", `the goods you want to by, find it from JD website. 
	Single Goods:
		produceID(:expectNum:expectPrice)
	Multiple Goods:
		produceID(:expectNum:expectPrice),produceID(:expectNum:expectPrice)`)
)

func main() {
	flag.Parse()
	defer clog.Shutdown()

	gs := parseGoods(*goods)
	clog.Trace("[Area: %+v, Goods: %+v, Period: %+v, Rush: %+v, Order: %+v]",
		*area, gs, *period, *rush, *order)

	jd := core.NewJingDong(core.JDConfig{
		Period:     time.Millisecond * time.Duration(*period),
		ShipArea:   *area,
		AutoRush:   *rush,
		AutoSubmit: *order,
	})

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGTSTP)
		<-signals
		signal.Stop(signals)
		jd.Release()

		os.Exit(0)
	}()

	if err := jd.Login(); err == nil {
		jd.CartDetails()
		fmt.Println()
		jd.RushBuy(gs)
	}
	jd.Release()

}

// parseGoods parse the input goods list. Support to input multiple goods sperated
// by comma(,). With an (:count) after goods ID to specify the count of each goods.
//
// Example as following:
//
//   2567304				single goods with default count 1, and any price
//   2567304:3				single goods with count 3, and any price
//   2567304,3133851:4		multiple goods with defferent count 1, 4, and any price
//   2567304:2:300,3133851:5:200	...
//
func parseGoods(goods string) []*core.ExpectProduct {
	lst := make([]*core.ExpectProduct, 0)
	if goods == "" {
		return lst
	}

	var err error
	for _, good := range strings.Split(goods, ",") {
		pair := strings.Split(good, ":")
		id := strings.Trim(pair[0], " ")
		num := 1
		if len(pair) > 1 {
			v, err := strconv.ParseInt(pair[1], 10, 32)
			if err != nil {
				panic(err)
			}
			num = int(v)
		}
		if num < 1 {
			panic("not a valid product count value")
		}

		price := math.MaxFloat64
		if len(pair) > 2 {
			price, err = strconv.ParseFloat(pair[2], 64)
			if err != nil {
				panic(err)
			}
		}
		if price < 0 {
			panic("not a valid price count value")
		}

		lst = append(lst, &core.ExpectProduct{
			ID:    id,
			Num:   num,
			Price: price,
		})
	}

	return lst
}

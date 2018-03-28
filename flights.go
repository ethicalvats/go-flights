package main

import (
	"encoding/json"
	"fmt"
	"os"

	"strconv"
	"time"

	"github.com/gocolly/colly"
)

// NoOfDays to look for
const NoOfDays int = 7

type flight struct {
	from string
	to   string
}

func main() {

	t := time.Now()
	f, _ := os.Create("log" + t.Format("02-01-2006-15-04-05") + ".txt")

	cities := [5]string{"BLR", "CCU", "MAA", "DEL", "BOM"}
	destinations := [2]string{"GOI", "HYD"}

	flights := make(map[string]flight)
	for _, c := range cities {
		for _, d := range destinations {
			flights[c+"-"+d] = flight{
				c, d,
			}
		}
	}

	fmt.Println("Date is", t.Local())
	for _, v := range flights {
		f.WriteString("Flight " + v.from + " " + v.to + "\r\n")
		var prices []int
		for i := 1; i < NoOfDays; i++ {
			tt := t.AddDate(0, 0, i)
			fetch(v.from, v.to, f, tt.Format("02/01/2006"), &prices)
		}
		fmt.Println(prices)
	}
}

func fetch(from string, to string, log *os.File, date string, slice *[]int) {
	fmt.Println(`************************************************START*****************************************************************	
	`)
	fmt.Println("Flight search started!", from, to, date)

	c := colly.NewCollector(
		colly.AllowedDomains("www.expedia.co.in"),
		colly.CacheDir("./flights_cache"),
	)

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	c.OnError(func(_ *colly.Response, err error) {
		fmt.Println("Something went wrong:", err)
	})

	c.OnResponse(func(r *colly.Response) {
		log.WriteString("Received response " + strconv.Itoa(r.StatusCode) + "\r\n")
		// log.WriteString(string(r.Body) + "\r\n")
	})

	c.OnHTML("body", func(e *colly.HTMLElement) {

		var finalPrice string
		p := &finalPrice
		// 	var li = jQuery('#flight-listing-container').find('ul#flightModuleList li')
		// 	li.each(function(i, el){
		// 	console.log($(el).find('span[data-test-id=listing-price-dollars]').text())
		// })

		e.ForEach("script#cachedResultsJson", func(_ int, el *colly.HTMLElement) {
			fmt.Println("Found #flight-listing-container")
			str := []byte(el.DOM.Text())
			var result interface{}
			json.Unmarshal(str, &result)
			data := result.(map[string]interface{})
			jsonParser(data, p, log)
			if len(*p) > 0 {
				fmt.Println("Flight price", from, to, date, "is", *p)
				log.WriteString("price " + date + " is " + *p + "\r\n")
				price, _ := strconv.ParseFloat(*p, 64)
				*slice = append(*slice, int(price))
				fmt.Println(len(*slice), price)
			}
		})
	})

	c.OnScraped(func(r *colly.Response) {
		fmt.Println("Finished")
		fmt.Println(`************************************************END*******************************************************************
		`)
	})

	c.Visit("https://www.expedia.co.in/Flights-Search?trip=oneway&leg1=from:" + from + ",to:" + to + ",departure:" + date + "TANYT&passengers=children:0,adults:1,seniors:0,infantinlap:Y&mode=search")
}

func jsonParser(m map[string]interface{}, finalValue *string, fs *os.File) {

	for key, value := range m {
		switch vv := value.(type) {
		case string:
			// fmt.Println(key, "is string")
			if key == "cheapestRoundedUpPrice" {
				*finalValue = vv
			}
		case int:
			// fmt.Println(key, "is int")
			if key == "cheapestRoundedUpPrice" {
				*finalValue = strconv.Itoa(vv)
			}
		case float64:
			// fmt.Println(key, "is float64")
			if key == "cheapestRoundedUpPrice" {
				*finalValue = strconv.FormatFloat(vv, 'f', 2, 64)
			}
		case bool:
			// fmt.Println(key, "is bool")
			if key == "cheapestRoundedUpPrice" {
				*finalValue = strconv.FormatBool(vv)
			}
		case []interface{}:
			// fmt.Println(key, "is an array:")
			for _, u := range vv {
				// fmt.Println(i, u)
				d := u.(map[string]interface{})
				jsonParser(d, finalValue, fs)
			}
		case interface{}:
			// fmt.Println(key, "is an array:")
			d := value.(map[string]interface{})
			jsonParser(d, finalValue, fs)
		default:
			*finalValue = "Type Not defined "
		}
	}
}

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

type priceStruct struct {
	price int
	date  string
}

type carrier struct {
	departureTime string
	arrivalTime   string
	airlineName   string
	plane         string
	airlineCode   string
	flightNumber  string
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
		var prices []priceStruct
		for i := 1; i < NoOfDays; i++ {
			tt := t.AddDate(0, 0, i)
			fetchFlightPrices(v.from, v.to, f, tt.Format("02/01/2006"), &prices)
		}
		fmt.Println(prices)
		minPrice, avgPrice := minPriceAndDate(prices)
		fmt.Println("Minimum price for", v.from, "to", v.to, "is", minPrice.price, "on", minPrice.date)
		fmt.Println("Average price over", NoOfDays, "days is", avgPrice)
		fetchFlightDetails(v.from, v.to, f, minPrice.date)
	}
}

func minPriceAndDate(data []priceStruct) (priceStruct, int) {
	var min int
	var sum int
	var priceDate priceStruct
	for _, d := range data {
		if min == 0 {
			min = d.price
		}
		if d.price < min {
			min = d.price
			priceDate = priceStruct{
				min, d.date,
			}
		}
		sum += d.price
	}
	avg := sum/len(data) + 1
	return priceDate, avg
}

func fetchFlightPrices(from string, to string, log *os.File, date string, slice *[]priceStruct) {
	// fmt.Println(`************************************************START*****************************************************************
	// `)
	// fmt.Println("Flight search started!", from, to, date)

	c := colly.NewCollector(
		colly.AllowedDomains("www.expedia.co.in"),
		colly.CacheDir("./flights_cache"),
	)

	c.OnRequest(func(r *colly.Request) {
		// fmt.Println("Visiting", r.URL)
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
			// fmt.Println("Found #flight-listing-container")
			str := []byte(el.DOM.Text())
			var result interface{}
			json.Unmarshal(str, &result)
			data := result.(map[string]interface{})
			jsonParserPrice(data, p, log)
			if len(*p) > 0 {
				// fmt.Println("Flight price", from, to, date, "is", *p)
				log.WriteString("price " + date + " is " + *p + "\r\n")
				price, _ := strconv.ParseFloat(*p, 64)
				*slice = append(*slice, priceStruct{
					int(price), date,
				})
			}
		})
	})

	c.OnScraped(func(r *colly.Response) {
		// fmt.Println("Finished")
		// fmt.Println(`************************************************END*******************************************************************
		// `)
	})

	// https://www.expedia.com/Flights-Search?trip=oneway&leg1=from:{0},to:{1},departure:{2}TANYT&passengers=adults:1,children:0,seniors:0,infantinlap:Y&options=cabinclass%3Aeconomy&mode=search&origref=www.expedia.com

	c.Visit("https://www.expedia.co.in/Flights-Search?trip=oneway&leg1=from:" + from + ",to:" + to + ",departure:" + date + "TANYT&passengers=children:0,adults:1,seniors:0,infantinlap:Y&options=cabinclass%3Aeconomy&mode=search&origref=www.expedia.com")
}

func jsonParserPrice(m map[string]interface{}, finalValue *string, fs *os.File) {

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
				jsonParserPrice(d, finalValue, fs)
			}
		case interface{}:
			// fmt.Println(key, "is an array:")
			d := value.(map[string]interface{})
			jsonParserPrice(d, finalValue, fs)
		default:
			*finalValue = "Type Not defined "
		}
	}
}

func fetchFlightDetails(from string, to string, log *os.File, date string) {
	fmt.Println(`************************************************START*****************************************************************	
	`)
	fmt.Println("Flight details extraction started!", from, to, date)

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

		var superlatives []interface{}
		var legs []interface{}
		s := &superlatives
		l := &legs

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
			jsonParserDetails(data, l, s, log)
			CheapestPriceDetails(l, s, log)
		})
	})

	c.OnScraped(func(r *colly.Response) {
		fmt.Println("Finished")
		fmt.Println(`************************************************END*******************************************************************
		`)
	})

	c.Visit("https://www.expedia.co.in/Flights-Search?trip=oneway&leg1=from:" + from + ",to:" + to + ",departure:" + date + "TANYT&passengers=children:0,adults:1,seniors:0,infantinlap:Y&options=cabinclass%3Aeconomy&mode=search&origref=www.expedia.com")
}

func jsonParserDetails(m map[string]interface{}, l *[]interface{}, s *[]interface{}, fs *os.File) {

	for key, value := range m {
		switch vv := value.(type) {
		case string:
			// fmt.Println(key, "is string")
			if key == "content" {
				// fs.WriteString(vv + "\r\n")
				var str string
				str = vv
				var content interface{}
				json.Unmarshal([]byte(str), &content)
				jsonObj := content.(map[string]interface{})
				jsonParserDetails(jsonObj, l, s, fs)
			}
		case int:
			// fmt.Println(key, "is int")

		case float64:
			// fmt.Println(key, "is float64")

		case bool:
			// fmt.Println(key, "is bool")

		case []interface{}:

			for _, u := range vv {
				// fmt.Println(v, "is an object:")
				if key == "superlatives" {
					*s = append(*s, u)
				}
				switch vvv := u.(type) {
				case string:
					// fmt.Println(v, "is string")
				case []interface{}:
					for _, w := range vvv {
						d := w.(map[string]interface{})
						jsonParserDetails(d, l, s, fs)
					}
				}
			}
		case interface{}:
			if key == "legs" {
				*l = append(*l, value)
			}
			d := value.(map[string]interface{})
			jsonParserDetails(d, l, s, fs)
		default:
			// fmt.Println(key, "is of type", reflect.TypeOf(value))
		}
	}
}

// CheapestPriceDetails ...
func CheapestPriceDetails(l *[]interface{}, s *[]interface{}, fs *os.File) {
	var cheapest interface{}
	for _, obj := range *s {
		data := obj.(map[string]interface{})
		for k, v := range data {
			if k == "superlativeType" && v == "CHEAPEST" {
				cheapest = data
			}
		}
	}
	// fmt.Println(cheapest)

	var cheapestID string
	data := cheapest.(map[string]interface{})
	for k, v := range data {
		if k == "offer" {
			obj := v.(map[string]interface{})
			for i, j := range obj {
				if i == "legIds" {
					cheapestID = j.([]interface{})[0].(string)
				}
			}
		}
	}
	fmt.Println(cheapestID)

	var flightDetailsObj interface{}
	for _, obj := range *l {
		data := obj.(map[string]interface{})
		for k := range data {
			if k == cheapestID {
				flightDetailsObj = data
			}
		}
	}
	fmt.Println(flightDetailsObj)

}

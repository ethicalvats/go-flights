package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"golang.org/x/net/context"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/db"
	"github.com/gocolly/colly"
	"google.golang.org/api/option"
)

// NoOfDays to look for
const NoOfDays int = 91

type flight struct {
	from string
	to   string
}

type priceStruct struct {
	price int
	date  string
}

type carrier struct {
	departureDate string
	arrivalDate   string
	departureTime string
	arrivalTime   string
	airlineName   string
	plane         string
	airlineCode   string
	flightNumber  string
}

type AirbusType struct {
	Description string `json:"description,omitempty"`
	Name        string `json:"name,omitempty"`
}

type PlaceType struct {
	Airport string `json:"AIRPORT,omitempty"`
	City    string `json:"CITY,omitempty"`
	Iata    string `json:"IATA,omitempty"`
}

type ArrivalType struct {
	Place PlaceType `json:"place,omitempty"`
	Time  string    `json:"time,omitempty"`
}

type DepartureType struct {
	Place PlaceType `json:"place,omitempty"`
	Time  string    `json:"time,omitempty"`
}

type LayoverType struct {
	Airbus AirbusType `json:"airbus,omitempty"`
	End    string     `json:"end,omitempty"`
	Place  PlaceType  `json:"place,omitempty"`
	Start  string     `json:"start,omitempty"`
}

type SingleFlightType struct {
	Airbus     AirbusType    `json:"airbus,omitempty"`
	Arrival    ArrivalType   `json:"arrival,omitempty"`
	AvgPrice   string        `json:"avg_price,omitempty"`
	Category   string        `json:"category,omitempty"`
	Connecting bool          `json:"connecting,omitempty"`
	Date       string        `json:"date,omitempty"`
	Departure  DepartureType `json:"departure,omitempty"`
	Layover    LayoverType   `json:"layover,omitempty"`
	Price      string        `json:"price,omitempty"`
	URLSlug    string        `json:"url_slug,omitempty"`
}

func main() {
	citiPtr := flag.String("city", "BLR", "Departure City")
	flag.Parse()
	DbClient := InitFirebaseDb()
	t := time.Now()
	IST, _ := time.LoadLocation("Asia/Kolkata")
	logPath := "/var/log/flights/" + *citiPtr + t.Format("02-01-2006-15-04") + ".txt"
	_, fileError := os.Create(logPath)
	f, fileError := os.OpenFile(logPath, os.O_RDWR, 0644)
	fmt.Println(fileError)
	cities := [1]string{*citiPtr} // BLR, DEL, CCU, BOM, MAA
	destinations := [15]string{"DEL", "BOM", "GOI", "CCU", "HYD", "GAU", "PNQ", "IXE", "MAA", "COK", "PAT", "JAI", "AMD", "BBI", "LKO"}

	flights := make(map[string]flight)
	for _, c := range cities {
		for _, d := range destinations {
			flights[c+"-"+d] = flight{
				c, d,
			}
		}
	}

	fmt.Println("Date is", t.Local(), "Departure", *citiPtr)
	f.WriteString("Date is " + t.Local().String() + " Departure " + *citiPtr)
	for _, v := range flights {
		f.WriteString("Flight " + v.from + " " + v.to + "\r\n")
		var prices []priceStruct
		for i := 1; i < NoOfDays; i++ {
			tt := t.AddDate(0, 0, i)
			fetchFlightPrices(v.from, v.to, f, tt.Format("02/01/2006"), &prices)
		}
		// fmt.Println(prices)
		minPrice, avgPrice := minPriceAndDate(prices)
		// fmt.Println("Minimum price for", v.from, "to", v.to, "is", minPrice.price, "on", minPrice.date)
		// fmt.Println("Average price over", NoOfDays, "days is", avgPrice)

		var flightsData carrier
		var flightsDataPtr *carrier
		flightsDataPtr = &flightsData
		fetchFlightDetails(v.from, v.to, f, minPrice.date, flightsDataPtr)

		var cheapestFlight SingleFlightType
		var cheapestFlightAirbus AirbusType
		var cheapestFlightArrival ArrivalType
		var cheapestFlightDeparture DepartureType

		arrivalTime, _ := time.ParseInLocation("02/01/2006 15:04", flightsData.arrivalDate+" "+flightsData.arrivalTime, IST)
		cheapestFlightArrival = ArrivalType{
			Time: arrivalTime.UTC().Format("2006-01-02T15:04:05.000Z"),
		}
		departureTime, _ := time.ParseInLocation("02/01/2006 15:04", flightsData.departureDate+" "+flightsData.departureTime, IST)
		cheapestFlightDeparture = DepartureType{
			Time: departureTime.UTC().Format("2006-01-02T15:04:05.000Z"),
		}
		cheapestFlightAirbus = AirbusType{
			Name:        flightsData.airlineName + " " + flightsData.airlineCode + " " + flightsData.flightNumber,
			Description: flightsData.plane,
		}
		cheapestFlight = SingleFlightType{
			Airbus:    cheapestFlightAirbus,
			Arrival:   cheapestFlightArrival,
			Departure: cheapestFlightDeparture,
			AvgPrice:  strconv.Itoa(avgPrice),
			Date:      departureTime.UTC().Format("2006-01-02T15:04:05.000Z"),
			Price:     strconv.Itoa(minPrice.price),
		}

		UpdateFlightData(DbClient, cheapestFlight, v, f)
	}

	// {
	// 	"airbus" : {
	// 	  "description" : "A320 6E 361",
	// 	  "name" : "Indigo"
	// 	},
	// 	"arrival" : {
	// 	  "place" : {
	// 		"AIRPORT" : "Chhatrapati Shivaji International Airport",
	// 		"CITY" : "Mumbai",
	// 		"IATA" : "BOM"
	// 	  },
	// 	  "time" : "1970-01-01T11:50:00.000Z"
	// 	},
	// 	"avg_price" : "5211",
	// 	"category" : "-L1xhzfcXqMqeY015iYT",
	// 	"connecting" : false,
	// 	"date" : "2018-03-12T18:30:00.000Z",
	// 	"departure" : {
	// 	  "place" : {
	// 		"AIRPORT" : "Kempegowda International Airport",
	// 		"CITY" : "Bengaluru",
	// 		"IATA" : "BLR"
	// 	  },
	// 	  "time" : "1970-01-01T10:05:00.000Z"
	// 	},
	// 	"price" : "1882",
	// 	"url_slug" : "bengaluru-mumbai"
	//   }

	// ReadFlightsData(DbClient)

}

func minPriceAndDate(data []priceStruct) (priceStruct, int) {
	var min int
	var sum int
	var avg int
	var priceDate priceStruct
	for _, d := range data {
		if min == 0 {
			min = d.price
			priceDate = priceStruct{
				min, d.date,
			}
		}
		if d.price < min {
			min = d.price
			priceDate = priceStruct{
				min, d.date,
			}
		}
		sum += d.price
	}
	if sum == 0 {
		avg = 0
	} else {
		avg = sum/len(data) + 1
	}
	return priceDate, avg
}

func fetchFlightPrices(from string, to string, log *os.File, date string, slice *[]priceStruct) {
	// fmt.Println(`************************************************START*****************************************************************
	// `)
	// fmt.Println("Flight search started!", from, to, date)

	c := colly.NewCollector(
		colly.AllowedDomains("www.expedia.co.in"),
		// colly.CacheDir("./flights_cache"),
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
		case int:
			// fmt.Println(key, "is int")
		case float64:
			// fmt.Println(key, "is float64")
			if key == "cheapestRoundedUpPrice" {
				*finalValue = strconv.FormatFloat(vv, 'f', 2, 64)
			}
		case bool:
			// fmt.Println(key, "is bool")
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

func fetchFlightDetails(from string, to string, log *os.File, date string, dataPtr *carrier) {
	// fmt.Println(`************************************************START*****************************************************************
	// `)
	// fmt.Println("Flight details extraction started!", from, to, date)

	c := colly.NewCollector(
		colly.AllowedDomains("www.expedia.co.in"),
		// colly.CacheDir("./flights_cache"),
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

		var superlatives []interface{}
		var legs []interface{}
		s := &superlatives
		l := &legs

		e.ForEach("script#cachedResultsJson", func(_ int, el *colly.HTMLElement) {
			// fmt.Println("Found #flight-listing-container")
			str := []byte(el.DOM.Text())
			var result interface{}
			json.Unmarshal(str, &result)
			data := result.(map[string]interface{})
			jsonParserDetails(data, l, s, log)
			flightData := CheapestPriceDetails(l, s, log)
			*dataPtr = flightData
		})
	})

	c.OnScraped(func(r *colly.Response) {
		// fmt.Println("Finished")
		// fmt.Println(`************************************************END*******************************************************************
		// `)
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
func CheapestPriceDetails(l *[]interface{}, s *[]interface{}, fs *os.File) carrier {
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
	// fmt.Println(cheapestID)

	var flightDetailsObj interface{}
	for _, obj := range *l {
		data := obj.(map[string]interface{})
		for k, v := range data {
			if k == cheapestID {
				flightDetailsObj = v
			}
		}
	}
	// fmt.Println(flightDetailsObj)

	var cheapestFlight carrier
	cheapestFlightPtr := &cheapestFlight

	flightDetailsObjMap := flightDetailsObj.(map[string]interface{})
	for key, value := range flightDetailsObjMap {
		// fmt.Println(key)
		if key == "departureTime" {
			data := value.(map[string]interface{})
			for k, v := range data {
				if k == "date" {
					cheapestFlightPtr.departureDate = v.(string)
				}
				if k == "time" {
					cheapestFlightPtr.departureTime = v.(string)
				}
			}
		}
		if key == "arrivalTime" {
			data := value.(map[string]interface{})
			for k, v := range data {
				if k == "date" {
					cheapestFlightPtr.arrivalDate = v.(string)
				}
				if k == "time" {
					cheapestFlightPtr.arrivalTime = v.(string)
				}
			}
		}
		if key == "timeline" {
			data := value.([]interface{})
			for index, v := range data {
				if index == 0 {
					for k, d := range v.(map[string]interface{}) {
						if k == "carrier" {
							obj := d.(map[string]interface{})
							for i, j := range obj {
								if i == "flightNumber" {
									cheapestFlightPtr.flightNumber = j.(string)
								}
								if i == "airlineName" {
									cheapestFlightPtr.airlineName = j.(string)
								}
								if i == "airlineCode" {
									cheapestFlightPtr.airlineCode = j.(string)
								}
								if i == "plane" {
									cheapestFlightPtr.plane = j.(string)
								}
							}
						}
					}
				}
			}
		}
	}
	// fmt.Println(cheapestFlight)
	return cheapestFlight
}

func InitFirebaseDb() *db.Client {
	opt := option.WithCredentialsFile("/usr/local/ICFClub/serviceAccountKey.json")
	config := &firebase.Config{DatabaseURL: "https://icfclub-98db1.firebaseio.com/"}
	app, err := firebase.NewApp(context.Background(), config, opt)
	if err != nil {
		fmt.Errorf("error initializing app: %v", err)
	}

	client, FirbaseClientErr := app.Database(context.Background())
	if FirbaseClientErr != nil {
		log.Fatalln("Error initializing database client:", FirbaseClientErr)
	}
	return client
}

func ReadFlightsData(client *db.Client) {

	ref := client.NewRef("/")
	singleFlightRef := ref.Child("single-flight")

	flights, err2 := singleFlightRef.OrderByKey().GetOrdered(context.Background())
	if err2 != nil {
		log.Fatalln("Error getting value:", err2)
	}
	snapshot := make([]SingleFlightType, len(flights))
	for i, r := range flights {
		var d SingleFlightType
		if err := r.Unmarshal(&d); err != nil {
			log.Fatalln("Error unmarshaling result:", err)
		}
		fmt.Println(d)
		snapshot[i] = d
	}
	// fmt.Println(snapshot)
}

func UpdateFlightData(client *db.Client, data SingleFlightType, trip flight, fs *os.File) {
	fs.WriteString("Updating flight info.. " + trip.from + " " + trip.to)
	fs.WriteString(data.AvgPrice + " " + data.Date + " " + data.Price)

	ref := client.NewRef("single-flight")
	var flightRef *db.Ref

	switch trip.from {
	case "BLR":
		if trip.to == "DEL" {
			flightRef = ref.Child("/-L29O2n5YwVPP38dJ_aJ")
		} else if trip.to == "BOM" {
			flightRef = ref.Child("/-L29OdjDvwJPLGmk737j")
		} else if trip.to == "GOI" {
			flightRef = ref.Child("/-L29P-jd2sYaJca6ZVW8")
		} else if trip.to == "CCU" {
			flightRef = ref.Child("/-L29PL0rIQwUibxpfAOs")
		} else if trip.to == "HYD" {
			flightRef = ref.Child("/-L29PeEL0yihJjLKGB8_")
		} else if trip.to == "GAU" {
			flightRef = ref.Child("/-L29QDyUO1mt--ClQRqz")
		} else if trip.to == "PNQ" {
			flightRef = ref.Child("/-L29QXDXnxaWGu42__DQ")
		} else if trip.to == "IXE" {
			flightRef = ref.Child("/-L29Qvay9Jv5mRUTS84S")
		} else if trip.to == "MAA" {
			flightRef = ref.Child("/-L29Rdbt8mmzgI03uCrJ")
		} else if trip.to == "COK" {
			flightRef = ref.Child("/-L29RzxP2caflO4I0WDq")
		} else if trip.to == "PAT" {
			flightRef = ref.Child("/-L29SVH-s7icttLWTjKk")
		} else if trip.to == "JAI" {
			flightRef = ref.Child("/-L29Syns5haRW5nObtDE")
		} else if trip.to == "AMD" {
			flightRef = ref.Child("/-L29TI1oZ6uNyDXNd9HZ")
		}
	case "CCU":
		if trip.to == "DEL" {
			flightRef = ref.Child("/-L29ZhIhgnhGvKxPW_d9")
		} else if trip.to == "MAA" {
			flightRef = ref.Child("/-L29ZxuDKVv9NEaPFKIC")
		} else if trip.to == "BLR" {
			flightRef = ref.Child("/-L29_e6dd9msrgv1A9ml")
		} else if trip.to == "BOM" {
			flightRef = ref.Child("/-L29aA60jfykFA22KqGR")
		} else if trip.to == "JAI" {
			flightRef = ref.Child("/-L29anYJw-p8SqoX3-90")
		} else if trip.to == "GOI" {
			flightRef = ref.Child("/-L29bEP-hBT7nyTYbADA")
		} else if trip.to == "HYD" {
			flightRef = ref.Child("/-L29bx-Ar7V9B5o2Qpsh")
		} else if trip.to == "GAU" {
			flightRef = ref.Child("/-L29clVkeoiOKULW4-or")
		} else if trip.to == "BBI" {
			flightRef = ref.Child("/-L29d1Mdt0vyStfpl2Dv")
		} else if trip.to == "PNQ" {
			flightRef = ref.Child("/-L29dNqLI0pSJmdXuK90")
		} else if trip.to == "PAT" {
			flightRef = ref.Child("/-L29df8myax0zNd_YgLl")
		} else if trip.to == "AMD" {
			flightRef = ref.Child("/-L29e5EuSk9TPED4Q_iV")
		}
	case "MAA":
		if trip.to == "DEL" {
			flightRef = ref.Child("/-L29gOxfU90MAvpuh-q3")
		} else if trip.to == "BOM" {
			flightRef = ref.Child("/-L29ggnij7eF6twHCkED")
		} else if trip.to == "GOI" {
			flightRef = ref.Child("/-L29P-jd2sYaJca6ZVW8")
		} else if trip.to == "CCU" {
			flightRef = ref.Child("/-L29h7ZukFmFTzMQLRYe")
		} else if trip.to == "BLR" {
			flightRef = ref.Child("/-L29hVIDmnKbh869eXNm")
		} else if trip.to == "HYD" {
			flightRef = ref.Child("/-L29hn-MvWSwOA8Rl2JH")
		} else if trip.to == "PNQ" {
			flightRef = ref.Child("/-L29i2eyg1NJwrE19fqE")
		}
	case "DEL":
		if trip.to == "GOI" {
			flightRef = ref.Child("/-L29kB6SjhgPTPs19Ufn")
		} else if trip.to == "BOM" {
			flightRef = ref.Child("/-L29kRaqw20D7xk85cWP")
		} else if trip.to == "BLR" {
			flightRef = ref.Child("/-L29kj7lz103h2Ucm51d")
		} else if trip.to == "MAA" {
			flightRef = ref.Child("/-L29l3YOwWLa1Zr9qADJ")
		} else if trip.to == "HYD" {
			flightRef = ref.Child("/-L29lKl5Iotc7FcGBa3p")
		} else if trip.to == "LKO" {
			flightRef = ref.Child("/-L29vlAgdUVoYJuLQahE")
		} else if trip.to == "PNQ" {
			flightRef = ref.Child("/-L29w2u1cQTJaWb9vpKg")
		}
	case "BOM":
		if trip.to == "DEL" {
			flightRef = ref.Child("/-L29wRe0WFr84rbAkmqu")
		} else if trip.to == "GOI" {
			flightRef = ref.Child("/-L2AOjAsQx8gage7E46_")
		} else if trip.to == "CCU" {
			flightRef = ref.Child("/-L2AP9P9lVZyiix7gGgK")
		} else if trip.to == "BLR" {
			flightRef = ref.Child("/-L2APSvH9seFFSi50rXL")
		} else if trip.to == "IXE" {
			flightRef = ref.Child("/-L2APlXzjsjd8x1WIney")
		} else if trip.to == "COK" {
			flightRef = ref.Child("/-L2AQKxJjzR5TTtWA57V")
		} else if trip.to == "MAA" {
			flightRef = ref.Child("/-L2AQaREzvfnDvmHM-DH")
		} else if trip.to == "HYD" {
			flightRef = ref.Child("/-L2AQqj_ioZkpEybO-50")
		} else if trip.to == "JAI" {
			flightRef = ref.Child("/-L2ARHRzeDu0gamMHcac")
		}
	}

	err := flightRef.Update(context.Background(), map[string]interface{}{
		"airbus":         data.Airbus,
		"arrival/time":   data.Arrival.Time,
		"avg_price":      data.AvgPrice,
		"date":           data.Date,
		"departure/time": data.Departure.Time,
		"price":          data.Price,
		"connecting":     false,
	})
	fmt.Println("Update error", err)
}

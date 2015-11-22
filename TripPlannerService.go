package main

import (
	// Standard library packages
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var TRACK_ID_CONSTANT int
var ACCESS_TOKEN string
var mgoSession *mgo.Session

type Message struct {
	Start_latitude  string `json:"start_latitude"`
	Start_longitude string `json:"start_longitude"`
	End_latitude    string `json:"end_latitude"`
	End_longitude   string `json:"end_longitude"`
	Product_id      string `json:"product_id"`
}

type PostRequest struct {
	Starting_from_location_id bson.ObjectId   `json:"id" bson:"id"`
	Location_ids              []bson.ObjectId `json:Location_ids`
}

type TripTrackerStrcuture struct {
	Tracker int `json:"tracker" "bson":"tracker"`
}

type Uberdata struct {
	End_id        bson.ObjectId
	Duration      float64
	Distance      float64
	High_Estimate float64
}

type TripPlannerResponse struct {
	Id                        string          `json:"id" "bson":"id"`
	Status                    string          `json:"status" "bson":"status"`
	Starting_from_location_id bson.ObjectId   `json:"starting_from_location_id" "bson":"starting_from_location_id"`
	Best_route_location_ids   []bson.ObjectId `json:"best_route_location_ids" "bson":"best_route_location_ids"`
	Total_uber_costs          float64         `json:"total_uber_costs" "bson":"total_uber_costs"`
	Total_uber_duration       float64         `json:"total_uber_duration" "bson":"total_uber_duration"`
	Total_distance            float64         `json:"total_distance" "bson":"total_distance"`
}

type PutTripPlannerResponse struct {
	// Id bson.ObjectId `json:"_id "bson:_id"`
	Id                           string          `json:"id" "bson":"id"`
	Status                       string          `json:"status" "bson":"status"`
	Starting_from_location_id    bson.ObjectId   `json:"starting_from_location_id" "bson":"starting_from_location_id"`
	Best_route_location_ids      []bson.ObjectId `json:"best_route_location_ids" "bson":"best_route_location_ids"`
	Uber_wait_time_eta           int             `json:"uber_wait_time_eta" "bson":"uber_wait_time_eta"`
	Current_location             bson.ObjectId   `json:"current_location" "bson":"current_location"`
	Next_destination_location_id bson.ObjectId   `json:"next_destination_location_id" "bson":"next_destination_location_id"`
	Total_uber_costs             float64         `json:"total_uber_costs" "bson":"total_uber_costs"`
	Total_uber_duration          float64         `json:"total_uber_duration" "bson":"total_uber_duration"`
	Total_distance               float64         `json:"total_distance" "bson":"total_distance"`
	// uber_wait_time_eta : 5

}

type PostResponse struct {
	Id           bson.ObjectId   `bson:"_id"`
	Name         string          `json:"name" bson:"name"`
	Address      string          `json:"address" bson:"address"`
	City         string          `json:"city" bson:"city"`
	State        string          `json:"state" bson:"state"`
	Zip          string          `json:"zip" bson:"zip"`
	Latitudes    interface{}     `json:"latitudes" bson:"latitudes"`
	Longitudes   interface{}     `json:"longitudes" bson:"longitudes"`
	Location_ids []bson.ObjectId `json:Location_ids`
}

func postHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	fmt.Println("-----------------SERVER LOGS ARE ENABLED--------------------------postHandler-------")

	var duration, total_uber_duration float64
	var distance, high_estimate, total_distance, total_uber_costs float64
	//var product_id string
	u := PostRequest{}
	var resp []PostResponse
	resp = append(resp, PostResponse{})

	json.NewDecoder(r.Body).Decode(&u)

	resp[0].Id = u.Starting_from_location_id
	resp[0].Location_ids = u.Location_ids

	resp = GetLocationListWithInfo(resp[0].Location_ids, resp[0].Id)
	var bestroute []PostResponse
	bestroute = append(bestroute, PostResponse{})
	bestroute = FindBestRoute(resp)
	var bestrouteLocationIds []bson.ObjectId

	for index, _ := range bestroute {
		bestrouteLocationIds = append(bestrouteLocationIds, bestroute[index].Id)
		startLocationlat := bestroute[index].Latitudes.(float64)
		startLocationlong := bestroute[index].Longitudes.(float64)

		if index != len(resp)-1 {
			endLocationlat := bestroute[index+1].Latitudes.(float64)
			endLocationlong := bestroute[index+1].Longitudes.(float64)

			duration, distance, high_estimate, _ = GetPriceEstimates(startLocationlat, startLocationlong, endLocationlat, endLocationlong)
			//	fmt.Println("This the product id",product_id)
		}

		if index == len(resp)-1 {

			duration, distance, high_estimate, _ = GetPriceEstimates(startLocationlat, startLocationlong, bestroute[0].Latitudes.(float64), bestroute[0].Longitudes.(float64))
			//	fmt.Println("This the product id",product_id)
		}
		total_uber_costs = total_uber_costs + high_estimate
		total_uber_duration = total_uber_duration + duration
		total_distance = total_distance + distance

	}

	resp[0].Location_ids = u.Location_ids
	//fmt.Println(resp)
	tripPlannerResponse := TripPlannerResponse{}
	rand.Seed(time.Now().UTC().UnixNano())
	TRACK_ID_CONSTANT = rand.Intn(5000)
	tripPlannerResponse.Id = strconv.Itoa(TRACK_ID_CONSTANT)
	tripPlannerResponse.Status = "planning"
	tripPlannerResponse.Starting_from_location_id = u.Starting_from_location_id
	bestrouteLocationIds = append(bestrouteLocationIds[:0], bestrouteLocationIds[1:]...)

	tripPlannerResponse.Best_route_location_ids = bestrouteLocationIds
	tripPlannerResponse.Total_uber_costs = total_uber_costs
	tripPlannerResponse.Total_uber_duration = total_uber_duration
	tripPlannerResponse.Total_distance = total_distance

	AttemptMongoInsertion(tripPlannerResponse)

	mgoSession, _ := mgo.Dial("mongodb://vinitgaikwad0810:Thisisvinit0810@ds043714.mongolab.com:43714/cmpe273-assignment2-mongodb")

	uj, _ := json.MarshalIndent(tripPlannerResponse, "", "\t")

	if err := mgoSession.DB("cmpe273-assignment2-mongodb").C("TripPlanner").Update(bson.M{"id": strconv.Itoa(TRACK_ID_CONSTANT)}, bson.M{"$set": bson.M{"tracker": 0}}); err != nil {
		fmt.Println("Insertion Failed")
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "The JSON response received is as follows %s", uj)

	fmt.Println("-----------------SERVER LOGS ARE ENABLED--------------------------postHandler-------")

}

func FindBestRoute(routes []PostResponse) []PostResponse { //

	var bestroute []PostResponse
	var uberdata []Uberdata
	bestroute = append(bestroute, PostResponse{})
	//uberdata = append(uberdata,Uberdata{})
	//uberdata[0].Start_id = routes[0].Id
	bestroute[0] = routes[0]
	routes = DeletefromOldRoute(routes, routes[0].Id)

	var NearbylocationId bson.ObjectId
	//fmt.Println(len(routes))
	lenroutes := len(routes)
	//fmt.Println("Length of the routes", routes)
	for i := 0; i < lenroutes; i++ {
		startLocationlat := bestroute[i].Latitudes.(float64)
		startLocationlong := bestroute[i].Longitudes.(float64)
		//fmt.Println("The bestroute Initially is",bestroute,"\n\n")
		uberdata = uberdata[:0]
		index_uber := 0
		for index, _ := range routes {

			endLocationlat := routes[index].Latitudes.(float64)
			endLocationlong := routes[index].Longitudes.(float64)
			uberdata = append(uberdata, Uberdata{})

			uberdata[index_uber].Duration, uberdata[index_uber].Distance, uberdata[index_uber].High_Estimate, _ = GetPriceEstimates(startLocationlat, startLocationlong, endLocationlat, endLocationlong)
			uberdata[index_uber].End_id = routes[index].Id
			//fmt.Println("These are the endids.......", uberdata[index1].End_id)
			index_uber++
			//fmt.Println("Uberdata High_Estimate", uberdata[index1].High_Estimate)

			//Delete(routes)
			//fmt.Println("This is the data",uberdata)

		}
		NearbylocationId = Returnlowest(uberdata)
		//fmt.Println("Returned Nearest Location is ", NearbylocationId)
		bestroute = CreationOfBestRoute(routes, NearbylocationId, bestroute)
		//fmt.Println("The Created bestroute  is", bestroute, "\n\n")
		routes = DeletefromOldRoute(routes, NearbylocationId)
		//fmt.Println("The New Routes Array after deletion is", routes, "\n\n")

	}
	//fmt.Println(uberdata)
	//fmt.Println("This is the data of bestroute------", bestroute, "\n\n")
	//return routes add later
	return bestroute
}

func CreationOfBestRoute(routes []PostResponse, NearbylocationId bson.ObjectId, bestroute []PostResponse) []PostResponse {
	index1 := len(bestroute)
	bestroute = append(bestroute, PostResponse{})
	for index, _ := range routes {
		if NearbylocationId == routes[index].Id {
			bestroute[index1] = routes[index]
		}
	}
	return bestroute
}

func DeletefromOldRoute(routes []PostResponse, NearbylocationId bson.ObjectId) []PostResponse {

	for index, _ := range routes {
		if NearbylocationId == routes[index].Id {
			routes = append(routes[:index], routes[index+1:]...)
			break
		}
	}
	return routes
}

func Returnlowest(uberdata []Uberdata) bson.ObjectId {
	min := 9999.00
	minduration := 9999.00
	var Id bson.ObjectId
	for index, _ := range uberdata {
		if uberdata[index].High_Estimate < min {
			min = uberdata[index].High_Estimate
			Id = uberdata[index].End_id
		} else if uberdata[index].High_Estimate == min {
			if uberdata[index].Duration < minduration {
				minduration = uberdata[index].Duration
				Id = uberdata[index].End_id
			}
		}

	}
	return Id
}

func GetLocationListWithInfo(location_ids []bson.ObjectId, starting_from_location_id bson.ObjectId) []PostResponse {
	var number []bson.ObjectId
	var startLocation bson.ObjectId
	startLocation = starting_from_location_id
	number = append(number, startLocation)
	var resp []PostResponse
	for index1, _ := range location_ids {
		fmt.Println("Location ID being considered is ", location_ids[index1])
	}

	//resp = append(resp, AttemptMongoConnection(startLocation))
	//resp := PostResponse{}
	// resp = AttemptMongoConnection(startLocation)
	for _, element := range location_ids {
		temp := element
		number = append(number, temp)
	}
	for index, _ := range number {
		resp = append(resp, AttemptMongoConnection(number[index]))
		temp_location := number[index]

		resp[index].Id = temp_location

	}

	return resp

}

func GetPriceEstimates(start_latitude float64, start_longitude float64, end_latitude float64, end_longitude float64) (float64, float64, float64, string) {
	var Url *url.URL
	Url, err := url.Parse("https://sandbox-api.uber.com")
	if err != nil {
		panic("Error Panic")
	}
	Url.Path += "/v1/estimates/price"
	parameters := url.Values{}
	start_lat := strconv.FormatFloat(start_latitude, 'f', 6, 64)
	start_long := strconv.FormatFloat(start_longitude, 'f', 6, 64)
	end_lat := strconv.FormatFloat(end_latitude, 'f', 6, 64)
	end_long := strconv.FormatFloat(end_longitude, 'f', 6, 64)
	parameters.Add("server_token", "5tyNL5jvocvFaQLfqGbZIyoB0xwMuQlJKVPr0l80")
	parameters.Add("start_latitude", start_lat)
	parameters.Add("start_longitude", start_long)
	parameters.Add("end_latitude", end_lat)
	parameters.Add("end_longitude", end_long)
	Url.RawQuery = parameters.Encode()

	res, err := http.Get(Url.String())
	//fmt.Println(Url.String())
	if err != nil {
		panic("Error Panic")
	}
	defer res.Body.Close()
	var v map[string]interface{}
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&v); err != nil {
		fmt.Println("ERROR: " + err.Error())
	}

	duration := v["prices"].([]interface{})[0].(map[string]interface{})["duration"].(float64)
	distance := v["prices"].([]interface{})[0].(map[string]interface{})["distance"].(float64)
	product_id := v["prices"].([]interface{})[0].(map[string]interface{})["product_id"].(string)
	fmt.Println("Product ID from Uber is ", product_id)
	high_estimate := v["prices"].([]interface{})[0].(map[string]interface{})["high_estimate"].(float64)

	return duration, distance, high_estimate, product_id
}

func AttemptMongoConnection(location bson.ObjectId) PostResponse {
	resp := PostResponse{}
	mgoSession, err := mgo.Dial("mongodb://vinitgaikwad0810:Thisisvinit0810@ds043714.mongolab.com:43714/cmpe273-assignment2-mongodb")

	if err != nil {
		fmt.Println("Connection Attempt Failed")
		panic(err)

	}
	//fmt.Println("Location int is ", location)

	// Fetch user
	if err := mgoSession.DB("cmpe273-assignment2-mongodb").C("UserLocation").FindId(location).One(&resp); err != nil {
		fmt.Println("Error Mongo DB")
		panic(err)
	}

	//fmt.Println("Returned Response ", resp)
	return resp
}

func getHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	fmt.Println("-----------------SERVER LOGS ARE ENABLED--------------------------putHandler-------")

	id := p.ByName("id")

	tripObject := TripPlannerResponse{}
	tripObject = AttemptMongoSelection(id)

	// Marshal provided interface into JSON structure

	// Write content-type, statuscode, payload

	uj, _ := json.MarshalIndent(tripObject, "", "\t")
	//fmt.Println(uj)
	// Write content-type, statuscode, payload
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "The JSON response received is as follows %s", uj)
	fmt.Println("-----------------SERVER LOGS ARE ENABLED--------------------------putHandler-------")
}

func AttemptMongoSelection(id string) TripPlannerResponse {
	resp := TripPlannerResponse{}

	mgoSession, err := mgo.Dial("mongodb://vinitgaikwad0810:Thisisvinit0810@ds043714.mongolab.com:43714/cmpe273-assignment2-mongodb")

	if err != nil {
		fmt.Println("Connection received")
		panic(err)

	}
	// Stub user

	// Fetch user
	if err := mgoSession.DB("cmpe273-assignment2-mongodb").C("TripPlanner").Find(bson.M{"id": id}).One(&resp); err != nil {
		//  w.WriteHeader(404)
		fmt.Println("Connection  recevied")
		panic(err)
	}
	return resp

}

func AttemptMongoInsertion(tripPlannerResponse TripPlannerResponse) {
	mgoSession, err := mgo.Dial("mongodb://vinitgaikwad0810:Thisisvinit0810@ds043714.mongolab.com:43714/cmpe273-assignment2-mongodb")

	if err != nil {
		panic(err)
	}
	mgoSession.DB("cmpe273-assignment2-mongodb").C("TripPlanner").Insert(tripPlannerResponse)
}

func GetETAFromUber(nextLocation bson.ObjectId, currentLocation bson.ObjectId) float64 {
	//ETA Retreiving
	var responseArray []PostResponse
	responseArray = append(responseArray, PostResponse{})
	responseArray = GetLocationListWithInfo([]bson.ObjectId{nextLocation}, currentLocation)
	startLocationlat := responseArray[0].Latitudes.(float64)
	startLocationlong := responseArray[0].Longitudes.(float64)
	endLocationlat := responseArray[1].Latitudes.(float64)
	endLocationlong := responseArray[1].Longitudes.(float64)

	_, _, _, product_id := GetPriceEstimates(startLocationlat, startLocationlong, endLocationlat, endLocationlong)

	v1 := Message{
		Start_latitude:  strconv.FormatFloat(startLocationlat, 'f', 6, 64),
		Start_longitude: strconv.FormatFloat(startLocationlong, 'f', 6, 64),
		End_latitude:    strconv.FormatFloat(endLocationlat, 'f', 6, 64),
		End_longitude:   strconv.FormatFloat(endLocationlong, 'f', 6, 64),
		Product_id:      product_id,
	}

	fmt.Println("This is the product id for the trip", product_id)

	jsonStr, _ := json.Marshal(v1)
	//fmt.Println(responseArray)
	client := &http.Client{}
	r, err := http.NewRequest("POST", "https://sandbox-api.uber.com/v1/requests", bytes.NewBuffer(jsonStr)) // <-- URL-

	if err != nil {
		panic(err)
	}

	r.Header.Set("Content-Type", "application/json")
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzY29wZXMiOlsicmVxdWVzdCJdLCJzdWIiOiJlN2RiOWQzYS05NDE3LTQzNzItOTdmOS0zNzU0Zjk3ODc3N2QiLCJpc3MiOiJ1YmVyLXVzMSIsImp0aSI6IjJlMmVkNDI5LTU3NjctNDBjMS05YzhiLWEwNzdkNTQ2MzljZiIsImV4cCI6MTQ1MDc1NDkyOCwiaWF0IjoxNDQ4MTYyOTI3LCJ1YWN0IjoiVGxHa2ZKbUY2enNLWEZpaXZjam1FbjlYMzhPcGRNIiwibmJmIjoxNDQ4MTYyODM3LCJhdWQiOiJmTEhJSjlpb3VnZmJPaVJBVEF1S2xIWVdWSDUyTkZHdiJ9.X03eAOCtuE6NaCV0nNeAOlz_F2ba0KdNpL1dYUN7kf8mUzM1NzbGzV4UQQyfzZTMuGK-cOworHTdRJyg_WyW0cHXjSnuzH7GNFbFaxCSsyt1ePPtT03gR3tgygkd-IL3zym57N_ADowb7DlGf3MeMIh81c_SI35n3PkdMXxRA0EBRfNaWQO6OiTvNJL3hrnx5F8mQ0tng3k3jiccJFx4rsjGCG7AwKk5WqcqNcDRdhtTCtrSFwjGiksb7F98DZn8hH6w2Rg8QAzmYRgrJkQ1HhoNLmawKpScJ0Fqa9H-CPX6pi5MX9J8uFo5GmG694dG5tfGvB4n6M7CduQH8n0Euw")

	resp, _ := client.Do(r)
	defer resp.Body.Close()
	//fmt.Println(r.Header)
	var v map[string]interface{}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&v); err != nil {
		fmt.Println("ERROR: " + err.Error())
	}

	//	fmt.Println("response Body:", v["eta"])

	return v["eta"].(float64)
}

func putHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	fmt.Println("-----------------SERVER LOGS ARE ENABLED--------------------------putHandler-------")

	var nextLocation, currentLocation bson.ObjectId

	tripOverFlag := 0

	id := p.ByName("id")
	trip_id, _ := strconv.Atoi(id)

	fmt.Println(trip_id)

	mgoSession, err := mgo.Dial("mongodb://vinitgaikwad0810:Thisisvinit0810@ds043714.mongolab.com:43714/cmpe273-assignment2-mongodb")
	if err != nil {
		panic(err)
	}

	//Retreiving the tracker

	tripTracker := TripTrackerStrcuture{}

	if err := mgoSession.DB("cmpe273-assignment2-mongodb").C("TripPlanner").Find(bson.M{"id": id}).One(&tripTracker); err != nil {
		fmt.Println("404 sent from here")
		w.WriteHeader(404)
		return
	}

	//fmt.Println("Tracking pointer is ", tripTracker.Tracker)

	putTripPlanner := PutTripPlannerResponse{}
	if err := mgoSession.DB("cmpe273-assignment2-mongodb").C("TripPlanner").Find(bson.M{"id": id}).One(&putTripPlanner); err != nil {
		fmt.Println("404 sent")
		w.WriteHeader(404)
		return
	}

	if tripTracker.Tracker == 0 {
		currentLocation = putTripPlanner.Starting_from_location_id
		nextLocation = putTripPlanner.Best_route_location_ids[0]
		if err := mgoSession.DB("cmpe273-assignment2-mongodb").C("TripPlanner").Update(bson.M{"id": id}, bson.M{"$set": bson.M{"current_location": currentLocation, "next_destination_location_id": nextLocation}}); err != nil {
			fmt.Println("404 sent")
			w.WriteHeader(404)
		}
		if err := mgoSession.DB("cmpe273-assignment2-mongodb").C("TripPlanner").Update(bson.M{"id": id}, bson.M{"$set": bson.M{"status": "requesting"}}); err != nil {
			fmt.Println("404 sent")
			w.WriteHeader(404)
			return
		}

		tripTracker.Tracker += 1

	} else {
		if tripTracker.Tracker == len(putTripPlanner.Best_route_location_ids) {
			currentLocation = putTripPlanner.Next_destination_location_id
			nextLocation = putTripPlanner.Starting_from_location_id

		} else if tripTracker.Tracker > len(putTripPlanner.Best_route_location_ids) {
			if err := mgoSession.DB("cmpe273-assignment2-mongodb").C("TripPlanner").Update(bson.M{"id": id}, bson.M{"$set": bson.M{"status": "completed"}}); err != nil {
				fmt.Println("404 sent")
				w.WriteHeader(404)
				return
			}
			tripOverFlag = 1
			fmt.Println("\n TRIP IS ALREADY OVER. CHECK THE STATUS \n")
			//Preparing the PUT RESPONSE
			//w.WriteHeader(404)
			fmt.Fprintf(w, "%s", "\n\n---------TRIP IS ALREADY OVER. CHECK THE STATUS----------\n\n")

			//	return
		} else {
			currentLocation = putTripPlanner.Next_destination_location_id
			nextLocation = putTripPlanner.Best_route_location_ids[tripTracker.Tracker]
		}
		if tripOverFlag != 1 {
			if err := mgoSession.DB("cmpe273-assignment2-mongodb").C("TripPlanner").Update(bson.M{"id": id}, bson.M{"$set": bson.M{"current_location": currentLocation, "next_destination_location_id": nextLocation}}); err != nil {
				fmt.Println("404 sent")
				w.WriteHeader(404)
				return
			}
		} else {

		}

		tripTracker.Tracker += 1

	}

	if err := mgoSession.DB("cmpe273-assignment2-mongodb").C("TripPlanner").Update(bson.M{"id": id}, bson.M{"$set": bson.M{"tracker": tripTracker.Tracker}}); err != nil {
		fmt.Println("Insertion Failed")
		panic(err)
	}
	var eta float64
	//ETA Retreiving
	//	fmt.Println("Trip Tracker", tripTracker.Tracker)
	length := len(putTripPlanner.Best_route_location_ids) + 1
	//fmt.Println("Length of the best route", length)

	if tripTracker.Tracker > length {
		eta = 0
	} else {
		eta = GetETAFromUber(nextLocation, currentLocation)
	}

	fmt.Println("ETA is", eta)

	if err := mgoSession.DB("cmpe273-assignment2-mongodb").C("TripPlanner").Update(bson.M{"id": id}, bson.M{"$set": bson.M{"uber_wait_time_eta": eta}}); err != nil {
		fmt.Println("Insertion Failed")
		panic(err)
	}

	//Preparing the PUT RESPONSE
	if err := mgoSession.DB("cmpe273-assignment2-mongodb").C("TripPlanner").Find(bson.M{"id": id}).One(&putTripPlanner); err != nil {
		fmt.Println("404 sent")
		w.WriteHeader(404)
		return
	}

	uj, _ := json.MarshalIndent(putTripPlanner, "", "\t")
	//fmt.Println(uj)
	// Write content-type, statuscode, payload
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "The JSON response received is as follows %s", uj)

	fmt.Println("-----------------SERVER LOGS ARE ENABLED--------------------------putHandler-------")

}

//curl -H 'Accept: application/json' -X PUT '{"Id":"3279","Location_ids": ["3528","1456"]}' http://localhost:8080/trips/1001/request
//curl -H 'Content-Type: application/json' -X PUT http://localhost:8080/trips/1001/request

func main() {

	fmt.Println("-----------------TRIP PLANNER SERVICE LISTENING--------------------------------------")
	TRACK_ID_CONSTANT = 1000
	ACCESS_TOKEN = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzY29wZXMiOlsicmVxdWVzdCJdLCJzdWIiOiJlN2RiOWQzYS05NDE3LTQzNzItOTdmOS0zNzU0Zjk3ODc3N2QiLCJpc3MiOiJ1YmVyLXVzMSIsImp0aSI6IjJlMmVkNDI5LTU3NjctNDBjMS05YzhiLWEwNzdkNTQ2MzljZiIsImV4cCI6MTQ1MDc1NDkyOCwiaWF0IjoxNDQ4MTYyOTI3LCJ1YWN0IjoiVGxHa2ZKbUY2enNLWEZpaXZjam1FbjlYMzhPcGRNIiwibmJmIjoxNDQ4MTYyODM3LCJhdWQiOiJmTEhJSjlpb3VnZmJPaVJBVEF1S2xIWVdWSDUyTkZHdiJ9.X03eAOCtuE6NaCV0nNeAOlz_F2ba0KdNpL1dYUN7kf8mUzM1NzbGzV4UQQyfzZTMuGK-cOworHTdRJyg_WyW0cHXjSnuzH7GNFbFaxCSsyt1ePPtT03gR3tgygkd-IL3zym57N_ADowb7DlGf3MeMIh81c_SI35n3PkdMXxRA0EBRfNaWQO6OiTvNJL3hrnx5F8mQ0tng3k3jiccJFx4rsjGCG7AwKk5WqcqNcDRdhtTCtrSFwjGiksb7F98DZn8hH6w2Rg8QAzmYRgrJkQ1HhoNLmawKpScJ0Fqa9H-CPX6pi5MX9J8uFo5GmG694dG5tfGvB4n6M7CduQH8n0Euw"
	r := httprouter.New()

	r.GET("/trips/:id", getHandler)
	r.POST("/trips", postHandler)
	r.PUT("/trips/:id/request", putHandler)

	server := http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: r,
	}
	server.ListenAndServe()
}

//56510d0cc2be633d74cdee5c - 475 West San Carlos Street
//56511459c2be6350f9b67de4 - "777 Story Rd
//565114c2c2be6350f9b67de5 - Adobe
//565114eec2be6350f9b67de6 - Paul Mitchel the School
//5651151cc2be6350f9b67de7 - 101 san fernando

//curl http://localhost:8080/trips/153
//curl -XPOST -H 'Content Type:application/json' -d '{"Id":"5651151cc2be6350f9b67de7","Location_ids": ["56510d0cc2be633d74cdee5c","56511459c2be6350f9b67de4","565114c2c2be6350f9b67de5","565114eec2be6350f9b67de6"]}' http://localhost:8080/trips
//curl -H 'Accept: application/json' -X PUT http://localhost:8080/trips/153/request

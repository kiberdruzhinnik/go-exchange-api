package errors

import "errors"

var ErrorNotFound = errors.New("not found")
var ErrorCouldNotFetchData = errors.New("could not fetch data")
var ErrorCouldNotParseJSON = errors.New("could not parse json")

var ErrorRedisNotConnected = errors.New("redis is not connected")
var ErrorRedisNotFound = errors.New("not found in redis")

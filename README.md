[![Build Status](http://img.shields.io/travis/o1egl/numcalc.svg?style=flat-square)](https://travis-ci.org/o1egl/numcalc)
[![Coverage Status](http://img.shields.io/coveralls/o1egl/numcalc.svg?style=flat-square)](https://coveralls.io/r/o1egl/numcalc)

Task
----

Write an HTTP service that exposes an endpoint "/numbers". This endpoint receives a list of URLs 
though GET query parameters. The parameter in use is called "u". It can appear 
more than once.

```http://yourserver:8080/numbers?u=http://example.com/primes&u=http://foobar.com/fibo```

When the /numbers is called, your service shall retrieve each of these URLs if 
they turn out to be syntactically valid URLs. Each URL will return a JSON data 
structure that looks like this:

```{ "numbers": [ 1, 2, 3, 5, 8, 13 ] }```

The JSON data structure will contain an object with a key named "numbers", and 
a value that is a list of integers. After retrieving each of these URLs, the 
service shall merge the integers coming from all URLs, sort them in ascending 
order, and make sure that each integer only appears once in the result. The 
endpoint shall then return a JSON data structure like in the example above with 
the result as the list of integers.

The endpoint needs to return the result as quickly as possible, but always 
within 500 milliseconds. It needs to be able to deal with error conditions when 
retrieving the URLs. If a URL takes too long to respond, it must be ignored. It 
is valid to return an empty list as result only if all URLs returned errors or 
took too long to respond.

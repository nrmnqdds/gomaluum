GoMa'luum
=========

i-Ma'luum scraper reimplementation with Go
------------------------------------------

<img src="https://github.com/nrmnqdds/simplified-imaluum/assets/65181897/2ad4fedc-1018-4779-b94a-5aae6f2944a3" width=100 />

🚧 **In Construction** 🚧
-------------------------

> [!IMPORTANT] 
> This project is **not** associated with the official i-Ma'luum!

Support this project!

[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/nrmnqdds)

A Reimplementation of the infamous [Simplified i-Ma'luum](https://imaluum.quddus.my) API in Go.

Swagger API documentation is available at [here](https://api.quddus.my/reference).

What's difference from previous version
---------------------------------------

-	[x] **Goroutine** for better concurrency performance
-	[x] **PASETO** for secure SSO token generation
-	[x] **gRPC** support for fast interservice communication
-	[x] **Docker** support

Local installation
------------------

> Requires go >= 1.23

```
git clone http://github.com/nrmnqdds/gomaluum
cd gomaluum
go mod tidy
air
```

Using Docker
------------

```
docker build -t gomaluum .
docker run -p 1323:1323 -d gomaluum
```

Todo
----

-	[ ] Scrape more data
-	[ ] Make it faster

GoMa'luum
=========

🚧 **In Construction** 🚧
-------------------------

> [!IMPORTANT] 
> This project is **not** associated with the official i-Ma'luum!

A proxy API which enables developers to build applications on top of i-Ma'luum.
Primarily used be some IIUM's student-made app:
- [Simplified i-Ma'luum](https://imaluum.quddus.my)
- [ProReg](https://proreg.app)

Swagger API documentation is available at [here](https://api.quddus.my/api/reference).

How it works under the hood
------------------------------------------

```mermaid
flowchart TD
    A["User sends request to GoMa'luum API"] --> B["GoMa'luum receives request"]
    B --> C["GoMa'luum sends request to original endpoint (imaluum.edu.my)"]
    C --> D["imaluum.edu.my returns auth cookie"]
    D --> E["GoMa'luum stores auth cookie, wraps with PASETO token"]
    E --> F["GoMa'luum uses cookie to scrape user data from imaluum.edu.my"]
    F --> G["GoMa'luum processes and formats scraped data"]
    G --> H["GoMa'luum returns pretty JSON/API response to user"]
    H --> I["User views data in a pretty UI or via API"]
    E --> J["PASETO token used for secure SSO/session management"]
    B --> K["Request validation, logging, and error handling"]
    F --> L["Concurrent scraping using goroutines for performance"]
    G --> M["Optional: Cache data for faster repeated access"]
    B --> N["gRPC support for internal/external service communication"]
```

<img src="https://github.com/nrmnqdds/simplified-imaluum/assets/65181897/2ad4fedc-1018-4779-b94a-5aae6f2944a3" width=100 />


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

Support this project!

[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/nrmnqdds)


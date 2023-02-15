
# Car rental system

My first project written in Go wtih Postgres. Simple web app that you can book a car in a few steps. The admin panel is used for creating new car offers, editing offers, listing and canceling reservations.


## Demo

Quick intro how it works.
![](https://github.com/gavronski/carRental/blob/main/intro-movie/carrental.gif)

Link onilne (sending mail is not set up on the server)

https://194-233-162-29.ip.linodeusercontent.com/

## Installation

Download the app 

```bash
  git clone https://github.com/gavronski/carRental.git
```
Add .env and database.yml files. Create postgres connection on your client and create the database compatible with settings.

Run migrations and seed tables with soda, from the root directory.

```bash
  soda migrate
```

Run the app. 
```bash
  go run ./cmd/web
```
or 

```bash
  ./app.exe
```


## Running Tests

To run unit tests, run the following command

```bash
  go test -v ./...
```


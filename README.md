
# Car rental system

My first project written in Go with Postgres. Layouts have been done with Bootstrap 5 and Sweetalert2. For database management, I've been using Soda CLI. Car rental is a simple web app that you can book a car in a few steps. The admin panel is done for creating new car offers, editing existing ones, listing and canceling reservations.


## Demo

Quick intro how it works.

![](https://github.com/gavronski/carRental/blob/main/intro-movie/carrental.gif)

Link onilne (sending mail is not set up on the server)



Credentials for the admin panel: 

email: admin@admin.com 
password: admin


## Installation

1. Download the app 

```bash
  git clone https://github.com/gavronski/carRental.git
```
2. Add .env and database.yml files. Create postgres connection on your client and create the database, compatible with settings.

3. Run migrations and seed tables with soda, from the "carRental" directory.

```bash
  soda migrate
```

4. Run the app. 
```bash
  go run ./cmd/web
```
or 

```bash
  ./app.exe
```
Make sure that you have installed and run MailHog (https://github.com/mailhog/MailHog/releases/v1.0.0).

## Running Tests

To run all unit tests, run the following command from "carRental" directory.

```bash
  go test -v ./...
```

To see tests coverage, change directory and run commands (for Windows):

```bash
  go test --coverprofile=coverage.out
  go tool cover --html=coverage.out
```


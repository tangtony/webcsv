# webcsv

webcsv is an application that automatically converts your CSV file into a SQLite database and provides an HTTP interface that allows you to run basic queries against the SQLite database and returns results in JSON format.

webcsv was designed to allow quick and easy integration of CSV datasets into your web stack.

## Installation

Download the latest binary from the [Releases](https://github.com/tangtony/webcsv/releases) page or pull the latest Docker image from [Docker Hub](https://cloud.docker.com/repository/docker/tangtony/webcsv).

## Usage
```
Usage: webcsv [options]
```

Run `webcsv` with the following options to provide information about your CSV. Options can be provided via environment variables or command-line. If both are provided, the command-line value takes precedence.

### Options
```
  --delimiter       The data separator used in the CSV file (default: ',') [$DELIMITER]
  --field-count     The number of fields/columns in the CSV file [$FIELD_COUNT]
  --file            The path to the CSV file [$FILE]
  --has-header      Whether the csv file has a header (default: true) [$HAS_HEADER]
  --header          A custom header to use for the data [$HEADER]
  --indicies        Headers to create indicies for [$INDICIES]
  --parse-numbers   Whether or not to parse JSON strings into numbers [$PARSE_NUMBERS]
```

NOTE: When using an escaped delimiter such as a tab, it must be specified as follows:
`--delimiter=$'\t'`

### Querying
```
http://localhost:8080?field=value
```

By default, `webcsv` starts a web server at port 8080 and querying for data is done at the root via query parameters where the key is the field/column that you want to match on and the value is the row value.

### Example

Below is an simple usage example of `webcsv`. 

Assume we want to serve [example.csv](example.csv):
```
firstName,lastName,email,phoneNumber
John,Doe,john@doe.com,0123456789
Jane,Doe,jane@doe.com,9876543210
```

The application can then be started with the following command:
```
./webcsv --file=example.csv
```

Once the application has been started, we can query for data as follows:
```
GET http://localhost:8080/?firstname=John
[
    {
        "firstname": "John",
        "lastname": "Doe"
    }
]
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details

## Contributing

1. Fork it (<https://github.com/tangtony/webcsv/fork>)
2. Create your feature branch (`git checkout -b feature/my-awesome-feature`)
3. Commit your changes (`git commit -am 'Add awesome feature'`)
4. Push to the branch (`git push origin feature/my-awesome-feature`)
5. Create a new Pull Request

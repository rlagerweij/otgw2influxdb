# OTGW2DB

OTGW2DB is a standalone program which can receive the output from the opentherm gateway ("OTGW") and pass it to InfluxDB for storage, analysis and graphing. It can also decode the message flow to human readable text for diagnostic purposes. OTGW2DB also contains a relay component which passes on the unmodified messages received to any connected client. 

The application is build in the Go programming language and can therefore be used on a great number of diffferent platforms including windows, macOs, linux, freebsd etc.

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

### Prerequisites

The current version of OTGW2DB is built with support for:
- InfluxDB 1.8 or higher with authentication turned on
- OTGW with a TCP/IP interface

Please check the InfluxDB website for details on how to install it. Setup a databse (or organisation & bucket) and a user with a password and the appropriate rights to write to the database. 

### Installing

OTGW2DB consists of a single executable and a configuration file. Just place the executable and the config file in the directory of your choice. Rename or copy the file otgw2db.exampe.cfg to otgw2db.cfg and edit its contents.

### The configuration file

The configuration file consists of two sections. The first section contains the settings which need to be adjusted to ensure that the program can connect to the OTGW and InfluxDB.

Currently the program can decode the otgw to either human readable form or the InfluxDB line protocol. 

The second part of the config file determines which opentherm messages will be decoded and stored. The Opentherm protocol contains many messages which contain static data (e.g. configuration settings) which is not very usefull to store in a time series database. The example config has a number of common usefull meatrics enabled for logging, but all opentherm messages can be enabled by changing the respective setting to "YES".

## First (Test) Run

After editting the configuration file it is recommended to run the program in verbose mode by starting it with the -v flag:

```
otgw2db-linux-arm64 -v
```

This will result in the program printing many of its intermediate steps to the console which allows you to check it is functioning correctly. For added information in the config file set: 

```
decode_readable = YES
```
After checking the program is decoding messages correctly you can stop execution with ctrl-C (^C)

## Start logging

For long term logging you should run the program in the background. otgw2db itself does not act as a daemon or windows service but there are many options to get the desired effect. 

Make sure that you do not run otgw2db in verbose or human readable mode in the background since it will fill up you storage with logs very quickly!

### Linux

The easiest way to run the program in the background on linux is to use the following command: 

```
otgw2db-linux-amd64 > otgw2db.log 2>&1 &
echo $! > otgw2db_pid.txt
```

These two commands run the program in the background and any output is stored in a file. 
otgw2db does not manage the size of these logs so keep an eye on them. Without the verbose option logging to file is minimal and limited to communication errors.

The second line stores the process id (pid) so that you can terminate the program at a later time by using the command:

```
kill -9 number_from_otgw2db_pid.txt
```

If you are starting the otgw2db over a remote (ssh) connection start it through nohup:

```
nohup otgw2db-linux-amd64 > otgw2db.log 2>&1 &
echo $! > otgw2db_pid.txt
```

### Windows

Windows Powershell supports running programs in the background through the [Start-Process](https://docs.microsoft.com/en-us/powershell/module/microsoft.powershell.management/start-process?view=powershell-7) cmdlet. 

Something like the command below should work but is not tested at this time. 

```
Start-Process -FilePath otgw2db-windows-amd64.exe -NoNewWindow -RedirectStandardOutput "otgw2db.log" -RedirectStandardError "otgw2dbError.log"
```

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details
# Horus_script

To run: 

docker build -t first . && docker run first sh -c "./main && ./create_users.sh"

This will run golang script and then it will create 100 users.

Or you can start docker container in interactive mode (-it) and execute golang program and script by yourself.

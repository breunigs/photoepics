```
cd dgraph && docker-compose up
# GUI is at http://localhost:8000/?local

# curl -X POST localhost:8080/alter -d '{"drop_all": true}'

go build
./photoepics gen --api-key <apikey> --filter-users <users> -i example.geojson --start-image <imgkey> --end-image <imgkey>
```



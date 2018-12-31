```
cd dgraph && docker-compose up
# GUI is at http://localhost:8000/?local

go vet ./...
go build

# Load data for a given file
# ./photoepics purge
./photoepics load --api-key <apikey> --filter-users <users> -i example.geojson

# Find image chains for previously loaded file
./photoepics query --start-image <imgkey> --end-image <imgkey>
```



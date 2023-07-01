# MapServer Wrapper

This repository contains a simple wrapper for MapServer, allowing cache of WMS tiles on disk.
It is completely written in Go.

## Build instructions

You can use Make to compile. Just run:

```make build```

Run the bootstrap command to copy the default dot file. Use the following command:

```make bootstrap```

## Enviroment variables

* **MAPSERVER_WRAPPER_MAP_SERV_EXEC_PATH**: Path to MapServer executable. Default is the MapServer executable located in this repository, in *bin/mapserv*;
* **MAPSERVER_WRAPPER_ALLOWED_ORIGINS**: API allowed origins, separated by commas (,). If not defined, default is *http://localhost:PORT*;
* **MAPSERVER_WRAPPER_PORT**: API HTTP port. Default is *8000*;
* **MAPSERVER_WRAPPER_BASE_PATH**: API base path. Default is */*;
* **MAPSERVER_WRAPPER_CACHE_PATH**: Directory where generated tiles are cached. Default is *cache* (located in the root of this repository).

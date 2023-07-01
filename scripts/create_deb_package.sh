#!/bin/bash

echo -e "Building Go app..."

make build

echo -e "Creating .deb package..."

mkdir -p deb/mapserver-wrapper_0.1_amd64/etc/mapserver-wrapper
mkdir -p deb/mapserver-wrapper_0.1_amd64/usr/bin
mkdir -p deb/mapserver-wrapper_0.1_amd64/tmp/mapserver-wrapper/cache

cp .env.example conf.env
mv conf.env deb/mapserver-wrapper_0.1_amd64/etc/mapserver-wrapper/conf.env

chmod 0755 deb/mapserver-wrapper_0.1_amd64
chmod 0755 deb/mapserver-wrapper_0.1_amd64/DEBIAN

rm -f deb/mapserver-wrapper_0.1_amd64/etc/mapserver-wrapper/VERSION
git rev-parse --verify HEAD >> deb/mapserver-wrapper_0.1_amd64/etc/mapserver-wrapper/VERSION

mv mapserver-wrapper deb/mapserver-wrapper_0.1_amd64/usr/bin

dpkg-deb --build deb/mapserver-wrapper_0.1_amd64
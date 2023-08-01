#!/usr/bin/env bash -e

PLUGIN_ID=$(grep '"id"' < src/plugin.json | sed -E 's/.*"id" *: *"(.*)".*/\1/')
echo "Validating the ${PLUGIN_ID} plugin"

echo "[Re]building the plugin"
rm -rf dist
npm run build

cleanup () {
  rm -rf ${PLUGIN_ID} ${PLUGIN_ID}.zip
}
trap cleanup EXIT


echo "Copying and zipping"
cleanup # in case stuff was left around from last time
cp -r dist ${PLUGIN_ID}
zip -qr ${PLUGIN_ID}.zip ${PLUGIN_ID}

echo "Invoking the plugin validator"
npx @grafana/plugin-validator@latest -sourceCodeUri file://. ${PLUGIN_ID}.zip

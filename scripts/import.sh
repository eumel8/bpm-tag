#!/usr/bin/env bash

# read mp3 files from current directory,
# give discnr as argument,
# extract metadata from mp3
# calculate bpm
# insert data into MySQL
#
MP3_DIR="."
DB_HOST="localhost"
DB_USER="musicuser"
DB_PASS="musicpass"
DB_NAME="DBmdb"

shopt -s nullglob

for file in "$MP3_DIR"/*.mp3; do

  discnr=${1:-0}
  meta=$(eyeD3 "$file")

  title=$(echo "$meta"  | awk -F': ' '/^title/  {print $2}')
  artist=$(echo "$meta" | awk -F': ' '/^artist/ {print $2}')

  # track: 3/12  -> 3
  pos=$(echo "$meta" | awk -F': ' '/^track/ {print $2}' | cut -d'/' -f1)
  pos=${pos:-0}


  # Time: 245 seconds
  timestr=$(echo "$meta" | awk -F': ' '/^Time/ {print $2}' | awk '{print $1}')
  # erwartet mm:ss oder hh:mm:ss
  IFS=: read -r t1 t2 t3 <<< "$timestr"

  if [[ -n "$t3" ]]; then
      # hh:mm:ss
      playtime=$((10#$t1*3600 + 10#$t2*60 + 10#$t3))
  elif [[ -n "$t2" ]]; then
      # mm:ss
      playtime=$((10#$t1*60 + 10#$t2))
  else
      playtime=0
  fi

  path=$(basename "$file")
  bpm=$(bpm-tag "$file")

echo $discnr $pos $title $artist $playtime $bpm $path

mysql -h${DB_HOST} -u${DB_USER} -p${DB_PASS} ${DB_NAME} <<SQL
INSERT INTO titel
(discnr, pos, playtime, bpm, title, artist, path)
VALUES
($discnr, $pos, $playtime, $bpm,
 '$(printf "%q" "$title")',
 '$(printf "%q" "$artist")',
 '$(printf "%q" "$path")'
);
SQL

done

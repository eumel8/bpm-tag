#!/usr/bin/env bash
#
# Shell Script um bpm tags zu Titel hinzuzufügen und
# fehlende Titel in der DB zu ersetzen

set -euo pipefail

# CREATE USER 'musicuser'@'localhost' IDENTIFIED by 'musicpass';
# GRANT SELECT, UPDATE ON DBmdb.titel TO 'musicuser'@'localhost';
# FLUSH PRIVILEGES;

# -----------------------------
# Konfiguration
# -----------------------------
DB_HOST="localhost"
DB_USER="musicuser"
DB_PASS="musicpass"
DB_NAME="DBmdb"


BASE_DIR="/widi/Media/Musik/Archive"
DB_DIR="/Media/Musik/Archive"

MYSQL="mysql -N -B -h${DB_HOST} -u${DB_USER} -p${DB_PASS} ${DB_NAME}"

# Alle Album-IDs aus der DB holen (numerisch)
${MYSQL} -e "SELECT DISTINCT discnr FROM titel ORDER BY discnr;" | while read -r DISCN_DB; do

    # DB -> Dateisystem (0001)
    DISCN_FS=$(printf "%04d" "${DISCN_DB}")
    ALBUM_DIR="${BASE_DIR}/${DISCN_FS}"


    if [[ ! -d "${ALBUM_DIR}" ]]; then
        echo "WARN: Album-Verzeichnis fehlt: ${ALBUM_DIR} (DB discnr=${DISCN_DB})"
        continue
    fi

    mapfile -t FILES < <(
        find "${ALBUM_DIR}" -maxdepth 1 -type f \
        \( -iname "*.mp3" -o -iname "*.flac" -o -iname "*.wav" \) \
        -printf "%f\n" | sort
    )

    TRACK_COUNT="${#FILES[@]}"
    if [[ "${TRACK_COUNT}" -eq 0 ]]; then
        echo "INFO: Keine Tracks für Album ${DISCN_DB}"
        continue
    fi

    POS=1
    for FILE in "${FILES[@]}"; do

        BASENAME="$(basename "$FILE")"
        NAME_NO_EXT="${BASENAME%.*}"
        NAME_NO_TRACK="${NAME_NO_EXT#[0-9][0-9] - }"

        TITLE_PARSED="${NAME_NO_TRACK%% - *}"
        ARTIST_PARSED="${NAME_NO_EXT##* - }"

        # SQL-escaping
        TITLE_ESC=$(printf "%s" "$TITLE_PARSED" | sed "s/'/''/g")
        ARTIST_ESC=$(printf "%s" "$ARTIST_PARSED" | sed "s/'/''/g")

        BPM=$(bpm-tag "${ALBUM_DIR}/${FILE}" || echo "0")
        FILE_ESCAPED=$(printf "%s%s%s" "${DB_DIR}/${DISCN_FS}/${FILE}" | sed "s/'/''/g")

        # bpm hinzufügen
        ${MYSQL} -e "
            UPDATE titel
            SET path='${FILE_ESCAPED}', bpm='${BPM}'
            WHERE discnr=${DISCN_DB}
              AND pos=${POS};
        "

        # titel updaten
        ${MYSQL} -e "
        UPDATE titel
        SET
            title  = '${TITLE_ESC}',
            artist = '${ARTIST_ESC}'
        WHERE discnr=${DISCN_DB}
          AND pos=${POS}
          AND title REGEXP '^Track';
        "

        ((POS++))
    done

    echo "OK: Album ${DISCN_DB} -> ${DISCN_FS} (${TRACK_COUNT} Tracks)"
done

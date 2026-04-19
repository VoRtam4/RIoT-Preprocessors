#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import xml.etree.ElementTree as ET
import os
import sys

INPUT_FILE  = "pls_FCD_241201.xml"
OUTPUT_FILE = "pls_FCD_241201_LiteVersion.xml"

def strip_ns(tag: str) -> str:
    return tag.split('}', 1)[-1]

def process(input_path: str, output_path: str):
    with open(output_path, 'wb') as out:
        out.write(b'<?xml version="1.0" encoding="UTF-8"?>\n')
        out.write(b'<predefinedLocations>\n')

        # iterparse s END-only, abychom měli celý podstrom při END tagu
        for _, elem in ET.iterparse(input_path, events=("end",)):
            # odstraníme namespace z tagu
            elem.tag = strip_ns(elem.tag)
            # odstraníme namespace i z jeho potomků
            for sub in elem.iter():
                sub.tag = strip_ns(sub.tag)

            # zpracujeme jen kontejner
            if elem.tag != "predefinedLocationContainer":
                continue

            # připravíme nový lite-kontejner
            attrs = {}
            if "id" in elem.attrib:
                attrs["id"] = elem.attrib["id"]
            if "version" in elem.attrib:
                attrs["version"] = elem.attrib["version"]
            lite = ET.Element("predefinedLocationContainer", attrs)

            # 1) TS kód
            for v in elem.findall(".//predefinedLocationName//value"):
                if v.get("lang") == "cs" and v.text:
                    n = ET.SubElement(lite, "predefinedLocationName")
                    vv = ET.SubElement(n, "value", {"lang": "cs"})
                    vv.text = v.text.strip()
                    break

            # 2) primární LCD
            prim = elem.find(".//alertCMethod2PrimaryPointLocation//specificLocation")
            if prim is not None and prim.text:
                p = ET.SubElement(lite, "alertCMethod2PrimaryPointLocation")
                sc = ET.SubElement(p, "specificLocation")
                sc.text = prim.text.strip()

            # 3) sekundární LCD
            sec = elem.find(".//alertCMethod2SecondaryPointLocation//specificLocation")
            if sec is not None and sec.text:
                p2 = ET.SubElement(lite, "alertCMethod2SecondaryPointLocation")
                sc2 = ET.SubElement(p2, "specificLocation")
                sc2.text = sec.text.strip()

            # 4) direction
            dirc = elem.find(".//alertCDirectionCoded")
            if dirc is not None and dirc.text:
                d = ET.SubElement(lite, "alertCDirectionCoded")
                d.text = dirc.text.strip()

            # 5) openlrLocationReferencePoint
            c1 = elem.find(".//openlrLocationReferencePoint//openlrCoordinate")
            if c1 is not None:
                o1 = ET.SubElement(lite, "openlrLocationReferencePoint")
                lat = c1.findtext("latitude")
                lon = c1.findtext("longitude")
                if lat:
                    e_lat = ET.SubElement(o1, "latitude"); e_lat.text = lat.strip()
                if lon:
                    e_lon = ET.SubElement(o1, "longitude"); e_lon.text = lon.strip()

            # 6) openlrLastLocationReferencePoint
            c2 = elem.find(".//openlrLastLocationReferencePoint//openlrCoordinate")
            if c2 is not None:
                o2 = ET.SubElement(lite, "openlrLastLocationReferencePoint")
                lat2 = c2.findtext("latitude")
                lon2 = c2.findtext("longitude")
                if lat2:
                    e_lat2 = ET.SubElement(o2, "latitude"); e_lat2.text = lat2.strip()
                if lon2:
                    e_lon2 = ET.SubElement(o2, "longitude"); e_lon2.text = lon2.strip()

            # zapíšeme tento lite-kontejner
            out.write(ET.tostring(lite, encoding='utf-8'))
            out.write(b'\n')

            # teprve teď uvolníme podstrom kontejneru
            elem.clear()

        out.write(b'</predefinedLocations>\n')

def main():
    if not os.path.isfile(INPUT_FILE):
        print(f"ERROR: vstupní soubor nenalezen: {INPUT_FILE}", file=sys.stderr)
        sys.exit(1)
    process(INPUT_FILE, OUTPUT_FILE)
    print(f"Lite-verze XML byla uložena do: {OUTPUT_FILE}")

if __name__ == "__main__":
    main()

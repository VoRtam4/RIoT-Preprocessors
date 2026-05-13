"""
@file parser.py
@brief Pomocné zpracování a validace DATEX II XML zpráv.

@author Dominik Vondruška
@ingroup riot_datex_downloader

@par Autorský podíl
- Dominik Vondruška: návrh a implementace celé funkcionality souboru.
"""
import xml.etree.ElementTree as ET

def parse_datex_xml(xml_data: str) -> dict:
    """
    Pokusí se zpracovat XML data ve formě řetězce a převést je na strukturu ElementTree.
    Vrací slovník s informací o úspěchu parsování a názvu kořenového tagu.
    V případě chyby vrací slovník s popisem chyby.
    """
    try:
        root = ET.fromstring(xml_data)
        return {"status": "parsed", "tag": root.tag}
    except Exception as e:
        return {"error": str(e)}
from ncclient import manager
from lxml import etree
import sys

# Connection details
DEVICE = {
    'host': '192.168.4.64',
    'port': 830,
    'username': 'sys-admin',
    'password': 'sys-admin',
    'hostkey_verify': False
}

def backup_config():
    try:
        with manager.connect(**DEVICE) as m:
            # 1. Retrieve the running configuration
            # 'source' can be 'running', 'startup', or 'candidate'
            print(f"Retrieving configuration from {DEVICE['host']}...")
            response = m.get_config(source='running')

            # 2. Convert raw XML string into an lxml element object
            # response.data_xml contains the XML inside the <data> tag
            xml_data = etree.fromstring(response.data_xml.encode())

            # 3. Beautify the XML
            # pretty_print=True adds the newlines and indentation
            beautified_xml = etree.tostring(xml_data, 
                                           pretty_print=True, 
                                           encoding='unicode')

            # 4. Save to a file
            filename = f"running_config_new_{DEVICE['host']}.xml"
            with open(filename, 'w') as f:
                f.write(beautified_xml)

            print(f"Successfully saved beautified config to: {filename}")

    except Exception as e:
        print(f"An error occurred: {e}")

if __name__ == "__main__":
    backup_config()

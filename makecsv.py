#Author:		Astennu
#Created:		08.10.23
#
#Updated by:	Astennu
#Last Updated:	10.10.23
#
#Description:
#Takes input from go program
#Checks if it get a match with the specified regex
#If yes: if not already it will create a file called output.csv and write the data to it

import re
import csv
import sys
import os
import json

#get input from go
input_string = sys.stdin.read()

#get config from json
json_file = 'config.json'
with open(json_file) as json_config:
    config = json.load(json_config)
pattern = config["regexPattern"]
headers = config["headers"]

#decode special signs
decoded_input = input_string.replace("Ã¼", "ü").replace("Ã¶","ö").replace("Ã¤","ä").replace("Ã—", "x").replace("Ã©", "é")

# Extracting relevant information using regular expressions
regex = re.compile(pattern, re.DOTALL)
match = regex.search(decoded_input)

if match:
    # Get the number of capturing groups in the regex
    num_groups = len(match.groups())

    # Extract data from the match dynamically
    data = [match.group(i).strip().replace('\n', ' ') if i != num_groups else match.group(i).replace('\n', ' ').strip() for i in range(1, num_groups + 1)]
    
    # Check if the CSV file already exists
    csv_filename = 'output.csv'
    is_file_exists = os.path.isfile(csv_filename)

    # Writing to a CSV file
    with open(csv_filename, 'a', newline='', encoding='utf-8') as csvfile:
        csv_writer = csv.writer(csvfile)

        # If the file doesn't exist, write the header
        if not is_file_exists:
            csv_writer.writerow(headers)

        # Append the new data
        csv_writer.writerow(data)
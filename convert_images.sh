#!/bin/bash

# Find all .jpg files in the static/images directory and convert them to .webp
find static/images -name "*.jpg" -print0 | while IFS= read -r -d $'\0' file; do
    # Get the filename without the extension
    filename_no_ext="${file%.jpg}"
    
    # Convert the image to .webp
    echo "Converting ${file} to ${filename_no_ext}.webp"
    cwebp -q 80 "${file}" -o "${filename_no_ext}.webp"
    
    # If the conversion was successful, delete the original .jpg file
    if [ $? -eq 0 ]; then
        echo "Deleting ${file}"
        rm "${file}"
    else
        echo "Error converting ${file}. Aborting."
        exit 1
    fi
done

echo "Image conversion to WebP complete."

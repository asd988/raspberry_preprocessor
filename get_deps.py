import requests
from os import path
from os.path import join
from http import HTTPStatus as Status
import time

cache_folder = ".rpb_cache"

owner = 'misode'
repo = 'mcmeta'
branch = 'summary'
file_path = 'versions/data.json'
url = f'https://raw.githubusercontent.com/{owner}/{repo}/{branch}/{file_path}'

version_file = join(cache_folder, "versions.json")
version_etag_file = join(cache_folder, "versions.etag")

def get_dependencies():
    headers = {}
    if path.exists(version_etag_file) and path.exists(version_file):
        with open(version_etag_file, 'r') as f:
            etag = f.read()
            headers['If-None-Match'] = etag
    response = requests.get(url, headers=headers)
    
    match response.status_code:
        case Status.NOT_MODIFIED:
            print("Dependencies are already up to date!")
        case Status.OK:
            etag = response.headers.get('ETag')
            with open(version_file, 'wb') as f:
                f.write(response.content)
            with open(version_etag_file, 'w') as f:
                f.write(etag)
        case _:
            print("A problem occured while fetching data.")


if __name__ == "__main__":
    start = time.time()
    print("downlading deps... (this might take years)")
    get_dependencies()
    end = time.time()
    print(f"this took {end - start}, WHAT")
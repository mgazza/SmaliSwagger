# SmaliSwagger

**SmaliSwagger** extracts API definitions from Smali files and converts them into a Swagger (OpenAPI) specification for easy documentation and analysis.

## Features
- Parses Smali files to detect API endpoints
- Extracts HTTP methods, paths, and request parameters
- Converts the extracted API into a Swagger (OpenAPI 2.0) specification
- Supports Retrofit annotations for method extraction
- Outputs a structured `swagger.json` file

## Installation
### Prerequisites
- Go 1.18+

### Build
To build SmaliSwagger from source, run:
```sh
git clone https://github.com/mgazza/SmaliSwagger.git
cd SmaliSwagger
go build -o smali-swagger
```
This will create an executable named `smali-swagger` in the project directory.

## Usage
### Command-line Arguments
```sh
./smali-swagger [options] <smali_directory>
```

#### Options:
| Option       | Description                                      | Default Value    |
|-------------|------------------------------------------------|----------------|
| `--path`    | Directory containing Smali files (alternative to positional argument) | `cwd` (current directory) |
| `--output`  | Path to the output Swagger JSON file           | `swagger.json` |

### Example Usage
#### Basic usage (current directory as Smali path)
```sh
./smali-swagger
```
#### Specify Smali directory
```sh
./smali-swagger /path/to/smali
```
#### Use `--path` and `--output`
```sh
./smali-swagger --path /path/to/smali --output extracted_api.json
```

## Decompiling APKs
To extract Smali files from an APK, you can use [Apktool](https://github.com/iBotPeaches/Apktool):

### Install Apktool
```sh
brew install apktool   # macOS
sudo apt install apktool  # Debian/Ubuntu
```

### Decompile an APK
```sh
apktool d myapp.apk -o myapp-smali
```
This will generate a directory containing Smali files, which you can then pass to SmaliSwagger.

## Installation
To install SmaliSwagger system-wide:
```sh
go install github.com/yourusername/SmaliSwagger@latest
```
This will place `smali-swagger` in your `$GOPATH/bin`.

## Contributing
We welcome contributions!

Please ensure your code follows Go best practices and includes tests where appropriate.

## License
SmaliSwagger is licensed under the **Apache 2.0 License**. See [LICENSE](LICENSE) for details.


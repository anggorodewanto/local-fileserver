# Local File Server

A simple HTTP file server for sharing files on your local network. This server provides a web interface for browsing files, uploading new files, and downloading existing ones.

## Features

- 📂 Browse files and folders with an intuitive web interface
- 📤 Upload files through the web interface
- 📥 Download files with a single click
- 🔍 Search functionality to quickly find files
- 🔒 Optional restriction to local network access only
- 📱 Mobile-friendly responsive design

## Screenshots

(Add screenshots of the application here)

## Installation

### Option 1: Download binary

Download the latest binary from the [releases page](https://github.com/anggorodewanto/local-fileserver/releases).

### Option 2: Build from source

```bash
# Clone the repository
git clone https://github.com/anggorodewanto/local-fileserver.git
cd local-fileserver

# Build the binary
go build -o local-fileserver

# Make it executable (on Unix systems)
chmod +x local-fileserver
```

### Option 3: Install with Go

```bash
go install github.com/anggorodewanto/local-fileserver@latest
```

## Usage

```bash
# Start with default settings (port 8080, serving ~/Downloads)
./local-fileserver

# Specify a different port
./local-fileserver -port 9000

# Specify a different directory to serve
./local-fileserver -dir /path/to/files

# Allow access from outside your local network
./local-fileserver -local=false

# Show help information
./local-fileserver -help

# Show version information
./local-fileserver -version
```

## Options

| Flag | Description | Default |
|------|-------------|---------|
| `-port` | Port to serve on | `8080` |
| `-dir` | Directory to serve files from | `~/Downloads` |
| `-local` | Restrict access to local network only | `true` |
| `-version` | Show version information | - |
| `-help` | Show help message | - |

## Security Considerations

By default, this server only accepts connections from the local network (localhost, 192.168.x.x, 10.x.x.x, etc.) to prevent unintended external access. If you need to allow access from the internet, use `-local=false` but be aware of the security implications:

- There is no authentication mechanism
- All files in the served directory will be accessible
- Anyone can upload files to your server

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
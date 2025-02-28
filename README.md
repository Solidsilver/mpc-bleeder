# mpc-bleeder

A CLI tool to add bleed area to MTG cards for printing on MakePlayingCards.com (MPC).

## About

This program was built to address the need for adding bleed area to custom Magic: The Gathering cards when printing them through MPC. Many online card generators don't include the necessary bleed, which can result in artwork being cut off during the printing process.

`mpc-bleeder` uses the dimensions provided on the MPC website to add the correct amount of padding to your card images, ensuring your artwork prints perfectly. It assumes your input images have no bleed added.

## Features

  * **Handles single files or entire folders:** Process one image at a time or batch process an entire directory of cards.
  * **Customizable output directory:** Specify where you want the modified images to be saved (defaults to `bleeder_out`).
  * **Automatic bleed calculation:**  Calculates the appropriate bleed based on the image resolution and MPC specifications.

## Usage

```sh
mpc-bleeder [flags] <input>

Flags:
-h, --help            show this help message and exit
-o, --output string   output directory (default "bleeder_out")

Args:
<input>  path to a single image file or a directory containing images
```

**Example:**

To add bleed to all images in the "cards" directory and save the output to "cards\_with\_bleed":
```sh
mpc-bleeder -o cards_with_bleed cards
```

## Installation

### Releases

1. Head over to the [Releases](https://github.com/Solidsilver/mpc-bleeder/releases) tab
2. Download the latest release for your OS/Arch
3. Extract the package and put the `mpc-bleeder` executible somewhere accessible
4. Use the instructions above to run.

### Build

1.  **Make sure you have Go installed.** You can download it from the official website: [https://golang.org/dl/](https://golang.org/dl/)
2.  **Clone this repository:** `git clone https://github.com/Solidsilver/mpc-bleeder`
3.  **Navigate to the project directory:** `cd mpc-bleeder`
4.  **Build the executable:** `just build` or `go build`
5.  Use the instructions above to run.

## Contributing

Contributions are welcome\! Feel free to open issues for bug reports or feature requests. Pull requests are also appreciated.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.


use clap::{CommandFactory, Parser};
use image::DynamicImage;
use std::{
    io::{BufReader, Cursor, IsTerminal, Read},
    path::Path,
    process,
};
use termion::{color, style};

#[derive(Parser, Debug)]
struct Args {
    img_path: Option<String>,

    #[clap(short = 'v', default_value_t = false)]
    verbose: bool,
}

fn load_image_from_path(img_path: &str) -> anyhow::Result<DynamicImage> {
    if !Path::new(&img_path).exists() {
        eprintln!("Error: File not exists at path {}", &img_path);
        process::exit(1);
    }

    let img = image::open(&img_path).expect("image open error");
    Ok(img)
}

fn load_image_from_stdin() -> anyhow::Result<DynamicImage> {
    let stdin = std::io::stdin().lock();

    // check any stdin pipe ?
    //  yes -> read data 
    //  no -> print help and bye
    if stdin.is_terminal() {
        Args::command().print_help()?;
        process::exit(1)
    }

    let mut reader = BufReader::new(stdin);

    let mut buf = Vec::new();
    reader.read_to_end(&mut buf)?;

    let img_reader = Cursor::new(buf);
    Ok(
        image::io::Reader::new(img_reader)
        .with_guessed_format()
        .expect("Failed to read image format")
        .decode()
        .expect("Failed to decode")
    )
}

fn main() {
    let args = Args::parse();
    let mut img = match args.img_path {
        Some(ref img_path) => {
            load_image_from_path(&img_path).expect("Failed to load image from file")
        }
        None => load_image_from_stdin().expect("Failed to load image from stdin"),
    };
    let terminal_size = termion::terminal_size().expect("Faild to query terminal size");

    // let display image as 80% or terminal width
    let n_width = (((terminal_size.0 as f32) / 2.0) * 0.8).round() as u32;
    let n_height = (terminal_size.1 as f32 * 2.0).round() as u32;
    img = img.resize(n_width, n_height, image::imageops::FilterType::Nearest);

    let rgb8_buf = img.to_rgb8();
    let width = img.width();
    let height = img.height();

    if args.verbose {
        println!("Args: {:?}", args);
        println!("Img: width={:?}, height={}", width, height);
        println!("Terminal Size: {:?}", terminal_size);
        println!("Display Size: {:?}", (n_width, n_height));
    }

    for y in 0..height {
        for x in 0..width {
            let p = rgb8_buf.get_pixel(x, y);
            let [r, g, b] = p.0;
            let display_col = color::Rgb(r, g, b);
            print!("{}  {}", color::Bg(display_col), style::Reset);
        }
        println!();
    }
}

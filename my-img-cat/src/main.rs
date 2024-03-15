use clap::Parser;
use std::{path::Path, process};
use termion::{color, style};

#[derive(Parser, Debug)]
struct Args {
    img_path: String,

    #[clap(short = 'v', default_value_t=false)]
    verbose: bool
}

fn main() {
    let args = Args::parse();
    if !Path::new(&args.img_path).exists() {
        eprintln!("Error: File not exists at path {}", &args.img_path);
        process::exit(1);
    }

    let mut img = image::open(&args.img_path).expect("image open error");
    let terminal_size = termion::terminal_size().expect("query terminal size error");

    // let display image as 80% or terminal width
    let n_width = (((terminal_size.0 as f32)  / 2.0) * 0.8).round() as u32;
    let n_height = (terminal_size.1 as f32 * 2.0).round() as u32;
    img = img.resize(
        n_width,
        n_height,
        image::imageops::FilterType::Nearest,
    );

    let rgb8_buf = img.to_rgb8();
    let width = img.width();
    let height = img.height();

    if args.verbose {
        println!("Args: {:?}", args);
        println!("Img: width={:?}, height={}", width, height);
        println!("Terminal Size: {:?}", terminal_size);
        println!("Display Size: {:?}", (n_width,n_height));    
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

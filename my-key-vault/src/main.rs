use std::io::{self, Write};

use env_logger::Env;
use my_key_vault::MyKeyVault;
use log::{info, error};


const DEFAULT_SAVE_PATH :&str = "vault.store";
const USAGE :&str = r#"Usage: 
exit: exit program
set <key> <val>: update key vault
get <key>: get value of key
save: save to disk
"#;

fn main() {
    let vault_file_path = &String::from(DEFAULT_SAVE_PATH);

    // default log level
    env_logger::init_from_env(Env::new().filter("RUST_LOG").default_filter_or("debug"));

    // load or create vault
    let res: Result<MyKeyVault, anyhow::Error> = MyKeyVault::load_from_file(vault_file_path);
    let mut key_vault_ins : MyKeyVault;
    match res {
        Err(_) => {
            info!("file not found. create new one");
            key_vault_ins =  MyKeyVault::new();
        }
        Ok(val) => {
            key_vault_ins =  val;
        }
    }

    print!("Enter passphrase: ");
    io::stdout().flush().unwrap();
    let mut passphrase = String::new();
    io::stdin().read_line(&mut passphrase).unwrap();
    let encryption_key = key_vault_ins.derive_key(&passphrase.trim().to_string());

    loop {
        let mut input = String::new();
        print!("vault> ");
        io::stdout().flush().unwrap();
        io::stdin().read_line(&mut input).unwrap();
        let parts: Vec<&str> = input.trim().split_whitespace().collect();

        match parts.as_slice() {
            ["set", key, val] => {
                println!("add or replace key: {}, val: {}", key, val);

                match key_vault_ins.add_secret( &encryption_key, &key.to_string(), &val.to_string()) {
                    Ok(_) => {println!("Ok")}
                    Err(err) => {error!("Error: {}",err)}
                };
                continue;
            },
            ["get", key] => {
                match key_vault_ins.get_secret(&encryption_key, &key.to_string()) {
                    Ok(v) => {println!("{}",v)}
                    Err(err) => {error!("Error: {}",err)}
                }
            }
            ["save"] => {
                match key_vault_ins.save_to_file(vault_file_path) {
                    Ok(_) => {println!("Ok")}
                    Err(err) => {error!("Error: {}",err)}
                }
                continue;
            }
            ["exit"] => {
                println!("bye!");
                break
            },
            ["test"] => {
                println!("{:X?}", 
                key_vault_ins.derive_key(&"this_is_my_key".to_string())
            );
                continue;
            }
            _ => {
                println!("{}",USAGE);
                continue;
            }
        }

    }
}

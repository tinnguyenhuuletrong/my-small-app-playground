use std::collections::HashMap;
use serde_derive::{Serialize, Deserialize};
use crypto::pbkdf2::pbkdf2;
use crypto::sha2::Sha256;
use crypto::hmac::Hmac;
use aes_gcm::KeyInit;
use aes_gcm::Aes256Gcm;
use aes_gcm::aead::{Aead};
use rand::Rng;
use std::fs;


#[derive(Debug, Serialize, Deserialize)]
pub struct MyKeyVault {
    salt: [u8; 16],
    data: HashMap<String, Vec<u8>>,
    version: u8
}

impl  MyKeyVault{
    pub fn new() -> Self {
        let salt: [u8; 16] = rand::thread_rng().gen();
        MyKeyVault {
            salt,
            data: HashMap::new(),
            version: 1
        }
    }

    pub fn load_from_file(path: &String) -> anyhow::Result<Self> {
        let data = fs::read(path)?;
        let ins :MyKeyVault = bincode::deserialize(&data)?;
        Ok(ins)
    }


    pub fn save_to_file(&self, path: &String) -> anyhow::Result<&Self> {
        let data = bincode::serialize(self)?;
        fs::write(path, data)?;
        Ok(self)
    }

    pub fn add_secret(&mut self,enc_key: &[u8;32],  key: &String, val: &String) -> anyhow::Result<()> {
        let cipher = Aes256Gcm::new_from_slice(enc_key).unwrap();
        let nonce: [u8; 12] = rand::thread_rng().gen();
        let enc_val = cipher.encrypt(&nonce.into(), val.as_bytes()).unwrap();

        let mut combined = nonce.to_vec();
        combined.extend(enc_val);
        self.data.insert(key.to_string(),  combined);
        Ok(())
    }

    pub fn get_secret(&self, enc_key: &[u8;32], key: &String)  -> anyhow::Result<String>  {
        if let Some(data) = self.data.get(key) {
            let cipher = Aes256Gcm::new_from_slice(enc_key).unwrap();
            let (nonce, enc_val) = data.split_at(12);
            if let Ok(val) = cipher.decrypt(nonce.into(), enc_val) {
                let str_val = String::from_utf8(val)?;
                Ok(str_val)
            } else  {
                Err(anyhow::anyhow!("Can not decode value"))
            }
        } else {
            Err(anyhow::anyhow!("Key '{}' not exists", key))
        }
    }

    pub fn derive_key(&self, pass_pharse: &String) -> [u8;32] {
        // Using pbkdf2 with Hmac and Sha256 to derive the key
        let mut key = [0u8; 32];
        let mut mac = Hmac::new(Sha256::new(), pass_pharse.as_bytes());
        pbkdf2(&mut mac, &self.salt, 10000, &mut key); // 10000 iterations
        key
    }
}
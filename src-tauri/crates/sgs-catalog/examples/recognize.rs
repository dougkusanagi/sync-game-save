//! cargo run -p sgs-catalog --example recognize -- catalog.db "E:\Games\Jogo"
use sgs_catalog::{recognize::recognize, Catalog};
use std::path::Path;
fn main() {
    let a: Vec<String> = std::env::args().collect();
    let cat = Catalog::open(Path::new(&a[1])).unwrap();
    let env = sgs_core::env::system_env(Some(a[2].clone().into()));
    let r = recognize(&cat, Path::new(&a[2]), &env, cfg!(windows)).unwrap();
    println!("nome: {} (catálogo {:?}, certeza {:?})", r.name, r.catalog_id, r.confidence);
    println!("exe: {:?}", r.exe);
    for c in &r.save_candidates {
        println!("save: {} -> {} existe={} arquivos={} marcado={}", c.logical, c.resolved.display(), c.exists, c.files, c.selected);
    }
    println!("alternativas: {:?}", r.alternatives);
}

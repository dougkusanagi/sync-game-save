//! Mede a montagem do catálogo: cargo run --release -p sgs-catalog --example measure -- manifest.yaml out.db
use sgs_catalog::{build, Catalog};
use std::{path::Path, time::Instant};

fn main() {
    let a: Vec<String> = std::env::args().collect();
    let t = Instant::now();
    let stats = build(Path::new(&a[1]), Path::new(&a[2])).unwrap();
    println!("montagem: {:?} ({} jogos, {} regras)", t.elapsed(), stats.games, stats.files);
    println!("tamanho: {} MB", std::fs::metadata(&a[2]).unwrap().len() / 1_000_000);
    let c = Catalog::open(Path::new(&a[2])).unwrap();
    for q in ["wi", "witcher", "dark souls", "stardew", "half-life", "cyberpunk", "mario", "elden ring", "terraria", "aggelos"] {
        let t = Instant::now();
        let n = c.search(q, 10).unwrap().len();
        println!("busca {q:?}: {n} resultados em {:?}", t.elapsed());
    }
    let t = Instant::now();
    let n = c.by_install_dir("Half-Life").unwrap().len();
    println!("por pasta: {n} em {:?}", t.elapsed());
}

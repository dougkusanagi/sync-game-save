use sgs_catalog::Catalog;
use std::path::Path;
fn main() {
    let a: Vec<String> = std::env::args().collect();
    let c = Catalog::open(Path::new(&a[1])).unwrap();
    let mut hits = c.by_install_dir(&a[2]).unwrap();
    if hits.is_empty() { hits = c.search(&a[2], 5).unwrap(); }
    for h in hits {
        println!("== {} (id {})", h.name, h.id);
        for p in c.save_paths(h.id).unwrap() { println!("  save: {} [{}] os={:?} store={:?}", p.path, p.tags, p.os, p.store); }
        for l in c.launch_paths(h.id).unwrap() { println!("  launch: {l}"); }
    }
}

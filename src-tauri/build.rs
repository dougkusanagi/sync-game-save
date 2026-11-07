use std::process::Command;

/// Número do build mostrado em "Sobre": atalho do commit e data do build.
fn numero_do_build() -> String {
    let commit = Command::new("git")
        .args(["rev-parse", "--short=7", "HEAD"])
        .output()
        .ok()
        .filter(|o| o.status.success())
        .map(|o| String::from_utf8_lossy(&o.stdout).trim().to_string())
        .filter(|s| !s.is_empty())
        .unwrap_or_else(|| "local".to_string());
    let data = Command::new("git")
        .args(["show", "-s", "--format=%cs", "HEAD"])
        .output()
        .ok()
        .filter(|o| o.status.success())
        .map(|o| String::from_utf8_lossy(&o.stdout).trim().to_string())
        .filter(|s| !s.is_empty())
        .unwrap_or_default();
    if data.is_empty() { commit } else { format!("{commit} ({data})") }
}

fn main() {
    println!("cargo:rustc-env=SGS_BUILD={}", numero_do_build());
    println!("cargo:rerun-if-changed=../.git/HEAD");
    tauri_build::build()
}

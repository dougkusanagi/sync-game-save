fn main() {
    let args: Vec<String> = std::env::args().skip(1).collect();
    let code = match sgs_cli::exec(args, &mut std::io::stdout(), &mut std::io::stdin().lock()) {
        Ok(code) => code,
        Err(msg) => {
            eprintln!("Erro: {msg}");
            1
        }
    };
    std::process::exit(code);
}

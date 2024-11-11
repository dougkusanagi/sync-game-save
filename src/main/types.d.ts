interface Game {
	exe: string;
	name: string;
	save_files: SavePath[];
}

interface SavePath {
	path: string;
	same_dir: boolean;
}

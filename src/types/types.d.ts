export interface Game {
	id: number;
	name: string;
	exe: string;
	save_files: SavePath[];
}

export interface SavePath {
	path: string;
	same_dir: boolean;
}

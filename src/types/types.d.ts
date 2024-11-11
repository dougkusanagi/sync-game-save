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

export interface GameYaml {
	cloud?: {
		steam?: boolean;
	};
	files?: {
		[path: string]: {
			tags?: string[];
			when?: Array<{ os?: string; bit?: number; store?: string }>;
		};
	};
	installDir?: {
		[name: string]: Record<string, unknown>;
	};
	launch?: {
		[path: string]: Array<{
			when: { os?: string; bit?: number; store?: string };
		}>;
	};
	steam?: {
		id: number;
	};
}

const steamUrlBuilder = {
	library: (objectId: string) =>
		`https://steamcdn-a.akamaihd.net/steam/apps/${objectId}/header.jpg`,
	libraryHero: (objectId: string) =>
		`https://steamcdn-a.akamaihd.net/steam/apps/${objectId}/library_hero.jpg`,
	logo: (objectId: string) =>
		`https://cdn.cloudflare.steamstatic.com/steam/apps/${objectId}/logo.png`,
	cover: (objectId: string) =>
		`https://cdn.cloudflare.steamstatic.com/steam/apps/${objectId}/library_600x900.jpg`,
	icon: (objectId: string, clientIcon: string) =>
		`https://cdn.cloudflare.steamstatic.com/steamcommunity/public/images/apps/${objectId}/${clientIcon}.ico`,
};

export default steamUrlBuilder;

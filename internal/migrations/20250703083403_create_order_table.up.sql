create table items (
	id	bigserial not null primary key,
	ChrtID     int,  
	Price      int,    
	Rid        varchar(256), 
	Name       varchar(128), 
	Sale       int,    
	Size       varchar(128), 
	TotalPrice int,    
	NmID       int,    
	Brand      varchar(128) 
);

create table payment (
	id	bigserial not null primary key,
	Transaction  varchar(256),
	Currency     varchar(128), 
	Provider     varchar(128),
	Amount       int    ,
	PaymentDt    int  ,  
	Bank         varchar(128),
	DeliveryCost int   ,
	GoodsTotal   int 
);

create table "orders" (
	id	bigserial not null primary key,
	OrderUID          varchar(128),  
	Entry             varchar(128), 
	delivery_id_fk  bigserial,
	InternalSignature varchar(128),  
	payment_id_fk        bigserial,
	Locale            varchar(128), 
	CustomerID        varchar(128), 
	TrackNumber       varchar(128),  
	DeliveryService   varchar(128), 
	Shardkey          varchar(128),  
	SmID              int,
	totalprice              int
);

create table delivery (
	id	bigserial not null primary key,
	name varchar(256),
	phone varchar(256),
	zip varchar(256),
	city varchar(256),
	address varchar(256),
	region varchar(256),
	email varchar(256)
);

create table "order_items" (
	id	bigserial not null primary key, 
	order_id_fk        bigserial,
	item_id_fk         bigserial
);

create table "cache" (
	id	bigserial not null primary key, 
	order_id	int8, 
	app_key        varchar(128)
);

ALTER TABLE public.orders ADD CONSTRAINT payment_id_fkey FOREIGN KEY (payment_id_fk) REFERENCES public.payment(id) on update no action on delete no action not valid;
ALTER TABLE public.order_items ADD CONSTRAINT order_id_fkey FOREIGN KEY (order_id_fk) REFERENCES public.orders(id) match simple on update no action on delete no action not valid;
ALTER TABLE public.orders ADD CONSTRAINT  delivery_id_fkey FOREIGN KEY (delivery_id_fk) REFERENCES public.delivery(id) on update no action on delete no action not valid;


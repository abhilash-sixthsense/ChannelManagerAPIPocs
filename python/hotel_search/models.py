from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy import Column, Integer, String, Float, Boolean, Date

Base = declarative_base()

class Property(Base):
    __tablename__ = "properties"
    id = Column(Integer, primary_key=True)
    name = Column(String)
    city = Column(String)
    latitude = Column(Float)
    longitude = Column(Float)
    max_guests = Column(Integer)

    # Amenities (30+)
    has_pool = Column(Boolean, default=False)
    has_bar = Column(Boolean, default=False)
    smoking_allowed = Column(Boolean, default=False)
    has_wifi = Column(Boolean, default=False)
    has_gym = Column(Boolean, default=False)
    has_spa = Column(Boolean, default=False)
    has_parking = Column(Boolean, default=False)
    has_aircon = Column(Boolean, default=False)
    has_kitchen = Column(Boolean, default=False)
    has_balcony = Column(Boolean, default=False)
    pet_friendly = Column(Boolean, default=False)
    near_beach = Column(Boolean, default=False)
    near_airport = Column(Boolean, default=False)
    family_friendly = Column(Boolean, default=False)
    romantic = Column(Boolean, default=False)
    business_ready = Column(Boolean, default=False)
    breakfast_included = Column(Boolean, default=False)
    free_cancellation = Column(Boolean, default=False)
    early_checkin = Column(Boolean, default=False)
    late_checkout = Column(Boolean, default=False)
    wheelchair_accessible = Column(Boolean, default=False)
    tv = Column(Boolean, default=False)
    minibar = Column(Boolean, default=False)
    room_service = Column(Boolean, default=False)
    non_smoking_rooms = Column(Boolean, default=False)

class Availability(Base):
    __tablename__ = "availability"
    id = Column(Integer, primary_key=True)
    property_id = Column(Integer)
    date = Column(Date)
    available = Column(Boolean, default=True)
